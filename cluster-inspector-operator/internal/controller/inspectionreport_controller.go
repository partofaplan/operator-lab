package controller

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aiopsv1 "github.com/partofaplan/operator-lab/api/v1"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// InspectionReportReconciler reconciles an InspectionReport object
type InspectionReportReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile runs the inspection logic for InspectionReport resources
func (r *InspectionReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("inspectionreport", req.NamespacedName)

	// 1. Fetch the InspectionReport instance
	var report aiopsv1.InspectionReport
	if err := r.Get(ctx, req.NamespacedName, &report); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("InspectionReport resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get InspectionReport")
		return ctrl.Result{}, err
	}

	// 2. Set dummy inspection results
	prompt := "Act as a Kubernetes expert. Given this message: 'All systems operational.' Provide a health summary and recommendations."
	response, err := queryOllama("llama3", prompt)
	if err != nil {
		log.Error(err, "Failed to get AI analysis from Ollama")
		response = "Error: AI analysis failed."
	}
	
	report.Status.Summary = "AI-generated analysis:"
	report.Status.Recommendations = []string{response}

	// 3. Requeue after 1 hour
	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

// SetupWithManager registers the controller with the manager
func (r *InspectionReportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiopsv1.InspectionReport{}).
		Named("inspectionreport").
		Complete(r)
}

func queryOllama(model, prompt string) (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	})

	resp, err := http.Post("http://ollama.ollama.svc.cluster.local:11434/api/generate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Response, nil
}

