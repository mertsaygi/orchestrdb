package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type PostgresAdapter struct{}

func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{}
}

func (p *PostgresAdapter) CreateDatabase(ctx context.Context, params CreateDatabaseParams) error {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		params.Host, params.Port, params.AdminUser, params.Password,
	)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("postgres connection error: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres ping error: %w", err)
	}

	query := fmt.Sprintf("CREATE DATABASE %s", pq.QuoteIdentifier(params.Name))
	if _, err := sqlDB.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("create database error: %w", err)
	}

	return nil
}
