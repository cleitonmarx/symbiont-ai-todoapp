package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/XSAM/otelsql"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// InitDB initializes the Postgres database connection and runs migrations.
type InitDB struct {
	db            *sql.DB
	skipMigration bool
	Logger        *log.Logger `resolve:""`
	DBUser        string      `config:"DB_USER"`
	DBPass        string      `config:"DB_PASS"`
	DBHost        string      `config:"DB_HOST"`
	DBPort        string      `config:"DB_PORT" default:"5432"`
	DBName        string      `config:"DB_NAME"`
}

// Initialize sets up the database connection and runs migrations and registers
// the *sql.DB in the dependency container.
func (di *InitDB) Initialize(ctx context.Context) (context.Context, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		di.DBUser,
		di.DBPass,
		di.DBHost,
		di.DBPort,
		di.DBName,
	)

	db, err := otelsql.Open("postgres", dsn, otelsql.WithAttributes(
		semconv.DBSystemNamePostgreSQL,
		semconv.DBNamespace(di.DBName),
	))
	if err != nil {
		return ctx, fmt.Errorf("failed to connect to database: %w", err)
	}
	di.db = db

	// Run migrations
	if !di.skipMigration {
		if err := di.runMigrations(); err != nil {
			return ctx, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	depend.Register(di.db)

	return ctx, nil
}

func (di *InitDB) runMigrations() error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	driver, err := postgres.WithInstance(di.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	di.Logger.Println("InitDB: migrations applied successfully")
	return nil
}

func (di *InitDB) Close() {
	if di.db != nil {
		if err := di.db.Close(); err != nil {
			di.Logger.Printf("InitDB: failed to close database connection: %v", err)
		}
	}
}
