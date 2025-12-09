package services

import (
	"context"
	"time"

	v1alpha1 "github.com/mertsaygi/orchestrdb/src/api/v1alpha1"
	"github.com/mertsaygi/orchestrdb/src/db"
)

type DatabaseService struct {
	adapter db.Adapter
}

func NewDatabaseService(adapter db.Adapter) *DatabaseService {
	return &DatabaseService{
		adapter: adapter,
	}
}

// adminUser and adminPassword are resolved (from spec or Secret) before calling this.
func (s *DatabaseService) EnsureDatabase(
	ctx context.Context,
	dbRes *v1alpha1.Database,
	adminUser string,
	adminPassword string,
) (bool, string) {
	params := db.CreateDatabaseParams{
		Host:      dbRes.Spec.Host,
		Port:      dbRes.Spec.Port,
		AdminUser: adminUser,
		Password:  adminPassword,
		Name:      dbRes.Spec.Name,
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
