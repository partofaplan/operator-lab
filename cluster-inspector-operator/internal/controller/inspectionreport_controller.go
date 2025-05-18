package controller

import (
	"context"
	"time"

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
	report.Status.Summary = "All systems operational."
	report.Status.Recommendations = []string{"No action needed."}

	if err := r.Status().Update(ctx, &report); err != nil {
		log.Error(err, "Failed to update InspectionReport status")
		return ctrl.Result{}, err
	}

	log.Info("Inspection complete. Report updated.")

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
