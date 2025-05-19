package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aiopsv1 "github.com/partofaplan/operator-lab/api/v1"
)

// InspectionReportReconciler reconciles an InspectionReport object
type InspectionReportReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *InspectionReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("inspectionreport", req.NamespacedName)

	var report aiopsv1.InspectionReport
	if err := r.Get(ctx, req.NamespacedName, &report); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("InspectionReport resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get InspectionReport")
		return ctrl.Result{}, err
	}

	// Fetch pods
	var pods corev1.PodList
	if err := r.List(ctx, &pods); err != nil {
		log.Error(err, "Failed to list pods")
	}
	podReport := ""
	for _, pod := range pods.Items {
		podReport += fmt.Sprintf("Pod %s (Namespace: %s): Phase=%s, Ready=%t\n",
			pod.Name, pod.Namespace, pod.Status.Phase, isPodReady(&pod))
	}

	// Fetch events
	var events corev1.EventList
	if err := r.List(ctx, &events); err != nil {
		log.Error(err, "Failed to list events")
	}
	eventReport := ""
	for _, e := range events.Items {
		eventReport += fmt.Sprintf("[%s] %s/%s - %s: %s\n",
			e.FirstTimestamp.Time.Format("2006-01-02 15:04:05"),
			e.InvolvedObject.Kind,
			e.InvolvedObject.Name,
			e.Reason,
			e.Message,
		)
	}

	// Fetch nodes
	var nodes corev1.NodeList
	if err := r.List(ctx, &nodes); err != nil {
		log.Error(err, "Failed to list nodes")
	}
	nodeReport := ""
	for _, node := range nodes.Items {
		ready := "Unknown"
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				ready = string(cond.Status)
				break
			}
		}
		nodeReport += fmt.Sprintf("Node %s: Ready=%s\n", node.Name, ready)
	}

	// Generate prompt
	prompt := fmt.Sprintf(`
You are a Kubernetes expert. Analyze the following cluster state and provide a high-level health summary and detailed recommendations.

--- Pods ---
%s

--- Events ---
%s

--- Nodes ---
%s

Output format:
- Summary: ...
- Recommendations:
  - ...
  - ...
`, podReport, eventReport, nodeReport)

	retryAttempts := 1
	timeoutSeconds := 10

	if report.Spec.RetryAttempts > 0 {
		retryAttempts = report.Spec.RetryAttempts
	}
	if report.Spec.TimeoutSeconds > 0 {
		timeoutSeconds = report.Spec.TimeoutSeconds
	}

	// 🔁 Call Ollama with retry + timeout
	response, err := queryOllamaWithRetry("llama3", prompt, timeoutSeconds, retryAttempts)
	if err != nil {
		log.Error(err, "AI analysis failed")
		response = "AI analysis failed. Please check connectivity to Ollama."
	}

	// Refetch the most recent version before updating
	if err := r.Get(ctx, req.NamespacedName, &report); err != nil {
		log.Error(err, "Failed to re-fetch InspectionReport before status update")
		return ctrl.Result{}, err
	}

	// Write results back to status
	report.Status.Summary = "Cluster inspection completed by AI."
	report.Status.Recommendations = []string{response}
	if err := r.Status().Update(ctx, &report); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Inspection complete. Report updated.")
	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

func (r *InspectionReportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiopsv1.InspectionReport{}).
		Named("inspectionreport").
		Complete(r)
}

func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func queryOllamaWithRetry(model, prompt string, timeoutSeconds, retryAttempts int) (string, error) {
	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	})

	url := "http://ollama.ollama.svc.cluster.local:11434/api/generate"

	var lastErr error
	for i := 0; i < retryAttempts; i++ {
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second) // Backoff delay
			continue
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)

		var result struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = err
			continue
		}

		return result.Response, nil
	}

	return "", fmt.Errorf("ollama request failed after %d attempts: %w", retryAttempts, lastErr)
}
