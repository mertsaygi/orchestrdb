package controllers

import (
	"context"
	"time"

	v1alpha1 "github.com/mertsaygi/orchestrdb/src/api/v1alpha1"
	"github.com/mertsaygi/orchestrdb/src/services"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// UserReconciler reconciles User resources.
type UserReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	UserService *services.UserService
}

func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var user v1alpha1.User
	if err := r.Get(ctx, req.NamespacedName, &user); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If being deleted, we currently do nothing (no DROP ROLE yet).
	if !user.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// -----------------------------------------------------------------
	// 1) Ensure generated Secret does NOT already exist
	// -----------------------------------------------------------------
	secNs := user.Spec.GeneratedSecret.Namespace
	if secNs == "" {
		secNs = user.Namespace
	}

	var existing corev1.Secret
	err := r.Get(ctx, types.NamespacedName{
		Name:      user.Spec.GeneratedSecret.Name,
		Namespace: secNs,
	}, &existing)

	if err == nil {
		// Secret already exists -> fail as requested
		msg := "generatedSecret already exists; refusing to overwrite"
		user.Status.Created = false
		user.Status.LastError = msg
		user.Status.UpdatedAt = time.Now().Format(time.RFC3339)
		_ = r.Status().Update(ctx, &user)

		logger.Error(nil, msg, "secret", user.Spec.GeneratedSecret.Name, "namespace", secNs)
		return ctrl.Result{}, nil
	} else if !apierrors.IsNotFound(err) {
		// Real error fetching Secret
		user.Status.Created = false
		user.Status.LastError = err.Error()
		user.Status.UpdatedAt = time.Now().Format(time.RFC3339)
		_ = r.Status().Update(ctx, &user)

		logger.Error(err, "failed to get generatedSecret")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// At this point Secret does not exist -> we create it.

	// -----------------------------------------------------------------
	// 2) Resolve admin credentials
	// -----------------------------------------------------------------
	adminUser, adminPassword, err := r.UserService.ResolveAdminCredentials(ctx, &user)
	if err != nil {
		user.Status.Created = false
		user.Status.LastError = err.Error()
		user.Status.UpdatedAt = time.Now().Format(time.RFC3339)
		_ = r.Status().Update(ctx, &user)

		logger.Error(err, "failed to resolve admin credentials")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// -----------------------------------------------------------------
	// 3) Generate a strong password for this user
	// -----------------------------------------------------------------
	generatedPassword, err := r.UserService.GeneratePassword(32)
	if err != nil {
		user.Status.Created = false
		user.Status.LastError = err.Error()
		user.Status.UpdatedAt = time.Now().Format(time.RFC3339)
		_ = r.Status().Update(ctx, &user)

		logger.Error(err, "failed to generate password")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// -----------------------------------------------------------------
	// 4) Create the Secret with username/password
	// -----------------------------------------------------------------
	secret := &corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      user.Spec.GeneratedSecret.Name,
			Namespace: secNs,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"username": user.Spec.Username,
			"password": generatedPassword,
		},
	}

	if err := r.Create(ctx, secret); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Safety: in case of race, behave like "already exists" semantics
			msg := "generatedSecret already exists during create"
			user.Status.Created = false
			user.Status.LastError = msg
			user.Status.UpdatedAt = time.Now().Format(time.RFC3339)
			_ = r.Status().Update(ctx, &user)

			logger.Error(err, msg)
			return ctrl.Result{}, nil
		}

		user.Status.Created = false
		user.Status.LastError = err.Error()
		user.Status.UpdatedAt = time.Now().Format(time.RFC3339)
		_ = r.Status().Update(ctx, &user)

		logger.Error(err, "failed to create generatedSecret")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// -----------------------------------------------------------------
	// 5) Ensure the user exists in the DB with correct privileges
	// -----------------------------------------------------------------
	created, errMsg := r.UserService.EnsureUser(ctx, &user, generatedPassword, adminUser, adminPassword)
	if errMsg != "" {
		logger.Error(nil, "EnsureUser failed", "error", errMsg)
	}

	if err := r.Status().Update(ctx, &user); err != nil {
		logger.Error(err, "failed to update User status")
		return ctrl.Result{}, err
	}

	if !created {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager registers the User controller with the manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.User{}).
		Complete(r)
}
