package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AdminSecretRef references a Secret containing admin credentials
// for connecting to the database server.
type AdminSecretRef struct {
	// Name of the Secret
	Name string `json:"name"`

	// Namespace of the Secret (optional, defaults to User's namespace if empty)
	Namespace string `json:"namespace,omitempty"`

	// Key inside the Secret for the admin username
	UserKey string `json:"userKey,omitempty"`

	// Key inside the Secret for the admin password
	PasswordKey string `json:"passwordKey,omitempty"`
}

// GeneratedSecret defines where the operator writes the generated username/password.
// The Secret MUST NOT exist before the User is created.
type GeneratedSecret struct {
	// Name of the Secret that will be created by the operator
	Name string `json:"name"`

	// Namespace of the Secret (optional, defaults to User's namespace if empty)
	Namespace string `json:"namespace,omitempty"`
}

// UserAccessRule describes access for a single database or instance.
type UserAccessRule struct {
	// Database name on the target instance.
	// May be empty if scope = "instance".
	DBName string `json:"dbName,omitempty"`

	// Role for this database. Defaults to readonly.
	// Allowed values: readonly, readwrite, owner.
	Role string `json:"role,omitempty"`

	// Scope of this access rule.
	// "database" -> database-level privileges
	// "instance" -> instance-level privileges
	Scope string `json:"scope,omitempty"`
}

// UserSpec defines the desired state of a User.
type UserSpec struct {
	// Target database server hostname or IP address.
	Host string `json:"host"`

	// Target database server port (e.g. 5432).
	Port int32 `json:"port"`

	// Admin user with permissions to create roles and grant privileges.
	// Optional when adminSecretRef is used.
	AdminUser string `json:"adminUser,omitempty"`

	// Admin password for development/testing only.
	// In production, prefer adminSecretRef.
	AdminPassword string `json:"adminPassword,omitempty"`

	// Reference to a Secret containing admin credentials.
	// If set, this takes precedence over AdminUser/AdminPassword.
	AdminSecretRef *AdminSecretRef `json:"adminSecretRef,omitempty"`

	// SSL mode used by the operator when connecting to the server.
	// Example values: disable, require, verify-ca, verify-full.
	SSLMode string `json:"sslMode,omitempty"`

	// Database-side username to create.
	Username string `json:"username"`

	// Secret where the operator will write username/password.
	// If the Secret already exists, the operator should fail.
	GeneratedSecret GeneratedSecret `json:"generatedSecret"`

	// List of access rules for this user. Each entry may target a
	// different database and role on the same instance.
	Access []UserAccessRule `json:"access"`
}

// UserStatus defines the observed state of a User.
type UserStatus struct {
	// Whether the user has been successfully created and granted privileges.
	Created bool `json:"created,omitempty"`

	// Last error message, if any.
	LastError string `json:"lastError,omitempty"`

	// Last time the resource was reconciled (RFC3339 format).
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// +kubebuilder:object:root=true

// User is the Schema for the database users API.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

// DeepCopyObject implements runtime.Object for User.
func (in *User) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(User)
	// Shallow copy all fields first
	*out = *in

	// Deep copy ObjectMeta
	out.ObjectMeta = *in.ObjectMeta.DeepCopy()

	// Deep copy slice fields inside Spec if needed
	if in.Spec.Access != nil {
		out.Spec.Access = make([]UserAccessRule, len(in.Spec.Access))
		copy(out.Spec.Access, in.Spec.Access)
	}

	return out
}

// DeepCopyObject implements runtime.Object for UserList.
func (in *UserList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(UserList)
	*out = *in

	// Deep copy ListMeta
	out.ListMeta = *in.ListMeta.DeepCopy()

	// Deep copy items slice
	if in.Items != nil {
		out.Items = make([]User, len(in.Items))
		for i := range in.Items {
			// Use DeepCopyObject on each item
			u := in.Items[i].DeepCopyObject().(*User)
			out.Items[i] = *u
		}
	}

	return out
}
