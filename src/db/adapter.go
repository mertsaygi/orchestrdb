package db

import (
	"context"
)

type CreateDatabaseParams struct {
	Host      string
	Port      int32
	AdminUser string
	Password  string
	Name      string
}

type Adapter interface {
	CreateDatabase(ctx context.Context, params CreateDatabaseParams) error
}
