package db

import (
	"context"
)

// CreateDatabaseParams contains all connection and database creation parameters.
type CreateDatabaseParams struct {
	Host      string
	Port      int32
	AdminUser string
	Password  string
	Name      string

	// SSLMode controls how the adapter connects to Postgres.
	// Examples: "disable", "require", "verify-ca", "verify-full".
	// If empty, the adapter will fall back to a safe default ("disable" or similar).
	SSLMode string
}

// Adapter defines the interface all DB backends must implement.
type Adapter interface {
	// CreateDatabase ensures that a database exists on the target server.
	// Implementations should be idempotent: if the database already exists,
	// they should NOT treat it as a fatal error.
	CreateDatabase(ctx context.Context, params CreateDatabaseParams) error
}
