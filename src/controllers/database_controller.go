package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1alpha1 "github.com/mertsaygi/orchestrdb/src/api/v1alpha1"
	"github.com/mertsaygi/orchestrdb/src/services"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DatabaseReconciler reconciles Database custom resources
type DatabaseReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Log             logr.Logger
	DatabaseService *services.DatabaseService
}

// Reconcile is called when a Database resource changes or periodically requeued.
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("database", req.NamespacedName)

	var dbRes v1alpha1.Database
	if err := r.Get(ctx, req.NamespacedName, &dbRes); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If the resource is being deleted, we currently do nothing.
	if !dbRes.ObjectMeta.DeletionTimestamp.IsZero() {
		// TODO: add optional cleanup logic (e.g., drop database) if desired.
		return ctrl.Result{}, nil
	}

	// -------------------------------------------------------------------------
	// Resolve admin credentials:
	// 1. Start with inline spec fields (AdminUser/AdminPassword)
	// 2. If adminSecretRef is set, override from Secret
	// -------------------------------------------------------------------------
	adminUser := dbRes.Spec.AdminUser
	adminPassword := dbRes.Spec.AdminPassword

	if dbRes.Spec.AdminSecretRef != nil {
		secRef := dbRes.Spec.AdminSecretRef

		var secret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{
			Name:      secRef.Name,
			Namespace: dbRes.Namespace, // Secret is expected to live in the same namespace
		}, &secret); err != nil {
			msg := "failed to get adminSecretRef Secret: " + err.Error()
			dbRes.Status.Created = false
			dbRes.Status.LastError = msg
			dbRes.Status.UpdatedAt = time.Now().Format(time.RFC3339)
			_ = r.Status().Update(ctx, &dbRes)

			log.Error(err, "failed to get adminSecretRef Secret",
				"secret", secRef.Name,
				"namespace", dbRes.Namespace)

			// Requeue so we can try again once the Secret exists or is fixed
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		userBytes, ok := secret.Data[secRef.UserKey]
		if !ok {
			msg := "userKey not found in adminSecretRef Secret"
			dbRes.Status.Created = false
			dbRes.Status.LastError = msg
			dbRes.Status.UpdatedAt = time.Now().Format(time.RFC3339)
			_ = r.Status().Update(ctx, &dbRes)

			log.Error(nil, msg, "secret", secRef.Name, "key", secRef.UserKey)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		passBytes, ok := secret.Data[secRef.PasswordKey]
		if !ok {
			msg := "passwordKey not found in adminSecretRef Secret"
			dbRes.Status.Created = false
			dbRes.Status.LastError = msg
			dbRes.Status.UpdatedAt = time.Now().Format(time.RFC3339)
			_ = r.Status().Update(ctx, &dbRes)

			log.Error(nil, msg, "secret", secRef.Name, "key", secRef.PasswordKey)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		adminUser = string(userBytes)
		adminPassword = string(passBytes)
	}

	// -------------------------------------------------------------------------
	// Call service layer to ensure database exists
	// -------------------------------------------------------------------------
	created, errMsg := r.DatabaseService.EnsureDatabase(ctx, &dbRes, adminUser, adminPassword)
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
		// If creation failed, requeue after a delay
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager registers the controller with the manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Database{}).
		Complete(r)
}
