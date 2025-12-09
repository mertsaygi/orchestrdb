package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// API group for the CRD
	GroupName = "orchestrdb.mertsaygi.net"
	// Version of this API
	Version = "v1alpha1"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   GroupName,
	Version: Version,
}

type SecretRef struct {
	// Name of the Secret in the same namespace as the Database resource
	Name string `json:"name"`

	// Key in the Secret data that contains the admin username
	UserKey string `json:"userKey"`

	// Key in the Secret data that contains the admin password
	PasswordKey string `json:"passwordKey"`
}

// DatabaseSpec: desired state of the Database CR
type DatabaseSpec struct {
	// Hostname or IP address of the target database server
	Host string `json:"host"`

	// Port number of the target database server (e.g., 5432)
	Port int32 `json:"port"`

	// Admin user with permissions to create databases (optional if adminSecretRef is used)
	AdminUser string `json:"adminUser,omitempty"`

	// Password for the admin user (optional if adminSecretRef is used)
	AdminPassword string `json:"adminPassword,omitempty"`

	// Name of the database to create
	Name string `json:"name"`

	// Reference to a Secret that contains admin user & password
	// If set, this takes precedence over AdminUser/AdminPassword fields.
	AdminSecretRef *SecretRef `json:"adminSecretRef,omitempty"`

	// SSLMode controls how the operator connects to the DB server.
	// Example values (Postgres-style): "disable", "require", "verify-ca", "verify-full".
	// If omitted, a sensible default is used by the service layer.
	SSLMode string `json:"sslMode,omitempty"`
}

// DatabaseStatus: observed state updated by the operator
type DatabaseStatus struct {
	// Whether the database has been successfully created
	Created bool `json:"created,omitempty"`

	// Last encountered error message, if any
	LastError string `json:"lastError,omitempty"`

	// Last time the resource was reconciled (RFC3339 format)
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// +kubebuilder:object:root=true
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Desired state
	Spec DatabaseSpec `json:"spec,omitempty"`

	// Observed state
	Status DatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Database `json:"items"`
}

// DeepCopyObject implements runtime.Object for Database
func (in *Database) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(Database)
	// shallow copy of all fields
	*out = *in

	// deep copy ObjectMeta (it has its own DeepCopy)
	out.ObjectMeta = *in.ObjectMeta.DeepCopy()

	return out
}

// DeepCopyObject implements runtime.Object for DatabaseList
func (in *DatabaseList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(DatabaseList)
	*out = *in

	// deep copy ListMeta
	out.ListMeta = *in.ListMeta.DeepCopy()

	// deep copy slice of items
	if in.Items != nil {
		out.Items = make([]Database, len(in.Items))
		for i := range in.Items {
			out.Items[i] = *in.Items[i].DeepCopyObject().(*Database)
		}
	}

	return out
}

// Register types into the global Kubernetes scheme
func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&Database{},
		&DatabaseList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
