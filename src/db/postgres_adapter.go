package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
)

// PostgresAdapter implements the Adapter interface for PostgreSQL.
type PostgresAdapter struct{}

// NewPostgresAdapter creates a new PostgresAdapter.
func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{}
}

func (p *PostgresAdapter) buildAdminConnString(host string, port int32, user, password, sslMode, dbName string) string {
	if sslMode == "" {
		sslMode = "disable"
	}
	if dbName == "" {
		dbName = "postgres"
	}

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host,
		port,
		user,
		password,
		dbName,
		sslMode,
	)
}

// CreateDatabase is unchanged from before (idempotent).
func (p *PostgresAdapter) CreateDatabase(ctx context.Context, params CreateDatabaseParams) error {
	dsn := p.buildAdminConnString(params.Host, params.Port, params.AdminUser, params.Password, params.SSLMode, "postgres")

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("postgres connect error: %w", err)
	}
	defer conn.Close(ctx)

	query := fmt.Sprintf(`CREATE DATABASE "%s"`, params.Name)

	_, err = conn.Exec(ctx, query)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "42P04" {
			// duplicate_database -> treat as success
			return nil
		}
		return fmt.Errorf("postgres create database error: %w", err)
	}

	return nil
}

// EnsureUser ensures that a role exists and has the given privileges.
func (p *PostgresAdapter) EnsureUser(ctx context.Context, params EnsureUserParams) error {
	// Connect to the instance (postgres db)
	dsn := p.buildAdminConnString(params.Host, params.Port, params.AdminUser, params.Password, params.SSLMode, "postgres")

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("postgres connect error: %w", err)
	}
	defer conn.Close(ctx)

	// 1) Ensure role exists with given password
	// Use DO block for idempotent create/alter.
	roleSQL := fmt.Sprintf(`
DO $$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = %q) THEN
      CREATE ROLE "%s" LOGIN PASSWORD %q;
   ELSE
      ALTER ROLE "%s" WITH LOGIN PASSWORD %q;
   END IF;
END
$$;
`, params.Username, params.Username, params.GeneratedPassword, params.Username, params.GeneratedPassword)

	if _, err := conn.Exec(ctx, roleSQL); err != nil {
		return fmt.Errorf("postgres ensure role error: %w", err)
	}

	// 2) Apply access rules
	for _, a := range params.Access {
		role := strings.ToLower(a.Role)
		if role == "" {
			role = "readonly"
		}
		scope := strings.ToLower(a.Scope)
		if scope == "" {
			scope = "database"
		}

		switch scope {
		case "database":
			if a.DBName == "" {
				// skip invalid rule
				continue
			}

			// Database-level grants require connecting to that database.
			dbDsn := p.buildAdminConnString(params.Host, params.Port, params.AdminUser, params.Password, params.SSLMode, a.DBName)
			dbConn, err := pgx.Connect(ctx, dbDsn)
			if err != nil {
				return fmt.Errorf("postgres connect to db %s error: %w", a.DBName, err)
			}

			// Basic privileges
			switch role {
			case "owner":
				// Grant all privileges on database
				// Note: ALTER DATABASE OWNER TO is more correct for true ownership;
				// here we grant typical privileges.
				_, err = conn.Exec(ctx, fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE "%s" TO "%s"`, a.DBName, params.Username))
				if err != nil {
					dbConn.Close(ctx)
					return fmt.Errorf("grant all on database %s error: %w", a.DBName, err)
				}
				fallthrough
			case "readwrite":
				// Connect + usage/select/modify on all tables in public schema
				_, err = conn.Exec(ctx, fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, a.DBName, params.Username))
				if err != nil {
					dbConn.Close(ctx)
					return fmt.Errorf("grant connect on %s error: %w", a.DBName, err)
				}
				_, err = dbConn.Exec(ctx, fmt.Sprintf(`GRANT USAGE, SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO "%s"`, params.Username))
				if err != nil {
					dbConn.Close(ctx)
					return fmt.Errorf("grant readwrite on %s.public error: %w", a.DBName, err)
				}
			case "readonly":
				_, err = conn.Exec(ctx, fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, a.DBName, params.Username))
				if err != nil {
					dbConn.Close(ctx)
					return fmt.Errorf("grant connect on %s error: %w", a.DBName, err)
				}
				_, err = dbConn.Exec(ctx, fmt.Sprintf(`GRANT USAGE, SELECT ON ALL TABLES IN SCHEMA public TO "%s"`, params.Username))
				if err != nil {
					dbConn.Close(ctx)
					return fmt.Errorf("grant readonly on %s.public error: %w", a.DBName, err)
				}
			default:
				dbConn.Close(ctx)
				return fmt.Errorf("unsupported role: %s", a.Role)
			}

			dbConn.Close(ctx)

		case "instance":
			// Instance-level access (simple example):
			// allow CONNECT on all existing databases by granting on each dbName if provided,
			// or just skip if dbName is empty.
			if a.DBName == "" {
				// nothing concrete to grant at instance level without listing DBs
				continue
			}
			_, err = conn.Exec(ctx, fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, a.DBName, params.Username))
			if err != nil {
				return fmt.Errorf("grant instance-scope connect on %s error: %w", a.DBName, err)
			}

		default:
			// unknown scope: ignore or fail. Here we fail.
			return fmt.Errorf("unsupported scope: %s", a.Scope)
		}
	}

	return nil
}
