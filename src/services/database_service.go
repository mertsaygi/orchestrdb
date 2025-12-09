package services

import (
	"context"
	"time"

	v1alpha1 "github.com/mertsaygi/orchestrdb/src/api/v1alpha1"
	"github.com/mertsaygi/orchestrdb/src/db"
)

// DatabaseService wraps the DB adapter and contains business logic for
// reconciling Database resources.
type DatabaseService struct {
	adapter db.Adapter
}

// NewDatabaseService creates a new DatabaseService with the given adapter.
func NewDatabaseService(adapter db.Adapter) *DatabaseService {
	return &DatabaseService{
		adapter: adapter,
	}
}

// EnsureDatabase ensures that the database described by dbRes exists on the
// target server. adminUser/adminPassword are resolved before calling this
// method (from inline spec or from a Secret).
func (s *DatabaseService) EnsureDatabase(
	ctx context.Context,
	dbRes *v1alpha1.Database,
	adminUser string,
	adminPassword string,
) (bool, string) {
	// For now we hardcode SSLMode to "require". If needed, this can be
	// made configurable via the CRD spec (e.g., dbRes.Spec.SSLMode).
	params := db.CreateDatabaseParams{
		Host:      dbRes.Spec.Host,
		Port:      dbRes.Spec.Port,
		AdminUser: adminUser,
		Password:  adminPassword,
		Name:      dbRes.Spec.Name,
		SSLMode:   "require",
	}

	err := s.adapter.CreateDatabase(ctx, params)
	if err != nil {
		dbRes.Status.Created = false
		dbRes.Status.LastError = err.Error()
		dbRes.Status.UpdatedAt = time.Now().Format(time.RFC3339)
		return false, err.Error()
	}

	dbRes.Status.Created = true
	dbRes.Status.LastError = ""
	dbRes.Status.UpdatedAt = time.Now().Format(time.RFC3339)
	return true, ""
}
