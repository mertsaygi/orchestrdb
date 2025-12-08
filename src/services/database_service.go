package services

import (
	"context"
	"time"

	"github.com/mertsaygi/orchestrdb/src/api/v1alpha1"
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

func (s *DatabaseService) EnsureDatabase(ctx context.Context, dbRes *v1alpha1.Database) (bool, string) {
	params := db.CreateDatabaseParams{
		Host:      dbRes.Spec.Host,
		Port:      dbRes.Spec.Port,
		AdminUser: dbRes.Spec.AdminUser,
		Password:  dbRes.Spec.AdminPassword,
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
