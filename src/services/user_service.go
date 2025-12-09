package services

import (
	"context"
	"crypto/rand"
	"time"

	v1alpha1 "github.com/mertsaygi/orchestrdb/src/api/v1alpha1"
	"github.com/mertsaygi/orchestrdb/src/db"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UserService struct {
	k8sClient client.Client
	adapter   db.Adapter
}

func NewUserService(k8sClient client.Client, adapter db.Adapter) *UserService {
	return &UserService{
		k8sClient: k8sClient,
		adapter:   adapter,
	}
}

// GeneratePassword creates a cryptographically random password.
func (s *UserService) GeneratePassword(length int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b), nil
}

// ResolveAdminCredentials resolves admin user/password for a User resource.
func (s *UserService) ResolveAdminCredentials(ctx context.Context, user *v1alpha1.User) (string, string, error) {
	// If AdminSecretRef is set, it takes precedence.
	if user.Spec.AdminSecretRef != nil && user.Spec.AdminSecretRef.Name != "" {
		secNs := user.Spec.AdminSecretRef.Namespace
		if secNs == "" {
			secNs = user.Namespace
		}

		var secret corev1.Secret
		if err := s.k8sClient.Get(ctx, types.NamespacedName{
			Name:      user.Spec.AdminSecretRef.Name,
			Namespace: secNs,
		}, &secret); err != nil {
			return "", "", err
		}

		userKey := user.Spec.AdminSecretRef.UserKey
		if userKey == "" {
			userKey = "username"
		}
		passKey := user.Spec.AdminSecretRef.PasswordKey
		if passKey == "" {
			passKey = "password"
		}

		uBytes, ok := secret.Data[userKey]
		if !ok {
			return "", "", apierrors.NewBadRequest("admin username key not found in adminSecretRef")
		}
		pBytes, ok := secret.Data[passKey]
		if !ok {
			return "", "", apierrors.NewBadRequest("admin password key not found in adminSecretRef")
		}

		return string(uBytes), string(pBytes), nil
	}

	// Fallback to inline adminUser/adminPassword.
	if user.Spec.AdminUser == "" || user.Spec.AdminPassword == "" {
		return "", "", apierrors.NewBadRequest("adminUser/adminPassword or adminSecretRef must be provided")
	}

	return user.Spec.AdminUser, user.Spec.AdminPassword, nil
}

// EnsureUser maps the User spec to adapter params and updates status.
func (s *UserService) EnsureUser(
	ctx context.Context,
	user *v1alpha1.User,
	generatedPassword string,
	adminUser string,
	adminPassword string,
) (bool, string) {
	sslMode := user.Spec.SSLMode
	if sslMode == "" {
		sslMode = "require"
	}

	access := make([]db.UserAccess, 0, len(user.Spec.Access))
	for _, a := range user.Spec.Access {
		role := a.Role
		if role == "" {
			role = "readonly"
		}
		scope := a.Scope
		if scope == "" {
			scope = "database"
		}
		access = append(access, db.UserAccess{
			DBName: a.DBName,
			Role:   role,
			Scope:  scope,
		})
	}

	params := db.EnsureUserParams{
		Host:              user.Spec.Host,
		Port:              user.Spec.Port,
		AdminUser:         adminUser,
		Password:          adminPassword,
		SSLMode:           sslMode,
		Username:          user.Spec.Username,
		GeneratedPassword: generatedPassword,
		Access:            access,
	}

	if err := s.adapter.EnsureUser(ctx, params); err != nil {
		user.Status.Created = false
		user.Status.LastError = err.Error()
		user.Status.UpdatedAt = time.Now().Format(time.RFC3339)
		return false, err.Error()
	}

	user.Status.Created = true
	user.Status.LastError = ""
	user.Status.UpdatedAt = time.Now().Format(time.RFC3339)
	return true, ""
}
