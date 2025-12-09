package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PostgresAdapter implements the Adapter interface for PostgreSQL.
type PostgresAdapter struct{}

// NewPostgresAdapter creates a new PostgresAdapter.
func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{}
}

// buildAdminConnString builds a DSN for connecting as an admin user.
// It honours the SSLMode passed in CreateDatabaseParams; if empty, it falls
// back to "disable" (or any default you prefer).
func (p *PostgresAdapter) buildAdminConnString(params CreateDatabaseParams) string {
	sslMode := params.SSLMode
	if sslMode == "" {
		// Default behaviour if not explicitly set.
		// You can change this to "require" if you want SSL by default.
		sslMode = "disable"
	}

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		params.Host,
		params.Port,
		params.AdminUser,
		params.Password,
		sslMode,
	)
}

// CreateDatabase connects to Postgres using admin credentials and ensures
// that the target database exists. It is implemented to be idempotent:
// if the database already exists, it is treated as success.
func (p *PostgresAdapter) CreateDatabase(ctx context.Context, params CreateDatabaseParams) error {
	dsn := p.buildAdminConnString(params)

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("postgres connect error: %w", err)
	}
	defer conn.Close(ctx)

	sql := fmt.Sprintf(`CREATE DATABASE "%s"`, params.Name)
	_, err = conn.Exec(ctx, sql)
	if err != nil {
		// 42P04 = duplicate_database (Postgres SQLSTATE)
		type pgError interface{ SQLState() string }

		if pe, ok := err.(pgError); ok && pe.SQLState() == "42P04" {
			// Database already exists → treat as success
			return nil
		}

		return fmt.Errorf("postgres create database error: %w", err)
	}

	return nil
}
