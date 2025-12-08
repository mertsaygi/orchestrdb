package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1alpha1 "github.com/mertsaygi/orchestrdb/src/api/v1alpha1"
	"github.com/mertsaygi/orchestrdb/src/services"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DatabaseReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Log             logr.Logger
	DatabaseService *services.DatabaseService
}

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("database", req.NamespacedName)

	var dbRes v1alpha1.Database
	if err := r.Get(ctx, req.NamespacedName, &dbRes); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !dbRes.ObjectMeta.DeletionTimestamp.IsZero() {
		// Deletion handling can be added later
		return ctrl.Result{}, nil
	}

	created, errMsg := r.DatabaseService.EnsureDatabase(ctx, &dbRes)
	if errMsg != "" {
		log.Error(nil, "EnsureDatabase failed", "error", errMsg)
	} else if created {
		log.Info("Database ensured/created", "name", dbRes.Spec.Name)
	}

	if err := r.Status().Update(ctx, &dbRes); err != nil {
		log.Error(err, "status update failed")
		return ctrl.Result{}, err
	}

	if !created {
		// Retry later if database creation failed
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Database{}).
		Complete(r)
}
