package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"fmt"
	"log"
	"strings"

	"github.com/DataDog/go-sqllexer"
	"github.com/XSAM/otelsql"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// InitDB initializes the Postgres database connection and runs migrations.
type InitDB struct {
	db                 *sql.DB
	metricRegistration metric.Registration
	skipMigration      bool
	Logger             *log.Logger `resolve:""`
	DBUser             string      `config:"DB_USER"`
	DBPass             string      `config:"DB_PASS"`
	DBHost             string      `config:"DB_HOST"`
	DBPort             string      `config:"DB_PORT" default:"5432"`
	DBName             string      `config:"DB_NAME"`
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

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	cfg.AfterConnect = func(ctx context.Context, pgconn *pgx.Conn) error {
		return pgxvector.RegisterTypes(ctx, pgconn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return ctx, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	dbSystemAttributes := otelsql.WithAttributes(
		semconv.DBSystemNamePostgreSQL,
		semconv.DBNamespace(di.DBName),
	)

	di.db = otelsql.OpenDB(
		stdlib.GetPoolConnector(pool),
		dbSystemAttributes,
		otelsql.WithInstrumentAttributesGetter(withQueryAttributes(di.Logger)),
	)

	di.metricRegistration, err = otelsql.RegisterDBStatsMetrics(
		di.db,
		dbSystemAttributes,
	)
	if err != nil {
		return ctx, fmt.Errorf("failed to register db stats metrics: %w", err)
	}

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
		if di.metricRegistration != nil {
			if err := di.metricRegistration.Unregister(); err != nil {
				di.Logger.Printf("InitDB: failed to unregister metric registration: %v", err)
			}
		}

	}
}

func withQueryAttributes(logger *log.Logger) func(ctx context.Context, method otelsql.Method, query string, args []driver.NamedValue) []attribute.KeyValue {
	return func(ctx context.Context, method otelsql.Method, query string, args []driver.NamedValue) []attribute.KeyValue {
		if method != otelsql.MethodConnQuery && method != otelsql.MethodConnExec {
			return nil
		}
		attib := []attribute.KeyValue{}

		operations, tables := extractSQLOperation(logger, query)
		if len(operations) > 0 {
			attib = append(attib, semconv.DBQuerySummary(fmt.Sprintf("%s %s", strings.Join(operations, ","), strings.Join(tables, ","))))
		}
		if len(tables) > 0 {
			attib = append(attib, semconv.DBCollectionName(strings.Join(tables, ",")))
		}

		return attib
	}
}

// extractSQLOperation extracts the primary SQL operation and target tables from a query.
func extractSQLOperation(logger *log.Logger, query string) ([]string, []string) {
	normalizer := sqllexer.NewNormalizer(
		sqllexer.WithCollectTables(true),
		sqllexer.WithCollectCommands(true),
		sqllexer.WithCollectComments(false),
	)

	_, meta, err := normalizer.Normalize(query)
	if err != nil {
		logger.Printf("Failed to extract SQL operation from query: %v", err)
		return nil, nil
	}

	return meta.Commands, meta.Tables
}
