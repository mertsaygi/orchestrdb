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

// UserAccess describes access to a single database/instance.
type UserAccess struct {
	DBName string
	Role   string
	Scope  string
}

// EnsureUserParams contains all parameters needed to create/update a DB user.
type EnsureUserParams struct {
	Host      string
	Port      int32
	AdminUser string
	Password  string
	SSLMode   string

	Username string
	// GeneratedPassword is the password that will be set for the user.
	GeneratedPassword string

	Access []UserAccess
}

// Adapter defines the interface all DB backends must implement.
type Adapter interface {
	// CreateDatabase ensures that a database exists on the target server.
	// Implementations should be idempotent.
	CreateDatabase(ctx context.Context, params CreateDatabaseParams) error

	// EnsureUser ensures that the given user exists and has the requested access.
	// Implementations should be idempotent: if the user already exists, they
	// should update the password and privileges accordingly.
	EnsureUser(ctx context.Context, params EnsureUserParams) error
}
