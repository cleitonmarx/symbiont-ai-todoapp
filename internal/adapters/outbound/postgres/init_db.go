package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

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
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const (
	defaultDBMaxOpenConns      = 50
	defaultDBMinConns          = 5
	defaultDBMaxIdleConns      = 25
	defaultDBConnMaxLifetime   = 30 * time.Minute
	defaultDBConnMaxIdleTime   = 5 * time.Minute
	defaultDBHealthCheckPeriod = 1 * time.Minute
)

type dbPoolSettings struct {
	MaxOpenConns      int
	MinConns          int
	MaxIdleConns      int
	ConnMaxLifetime   time.Duration
	ConnMaxIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// InitDB initializes the Postgres database connection and runs migrations.
type InitDB struct {
	SkipMigration       bool
	db                  *sql.DB
	metricRegistration  metric.Registration
	Logger              *log.Logger   `resolve:""`
	DBUser              string        `config:"DB_USER"`
	DBPass              string        `config:"DB_PASS"`
	DBHost              string        `config:"DB_HOST"`
	DBPort              string        `config:"DB_PORT" default:"5432"`
	DBName              string        `config:"DB_NAME"`
	DBMaxOpenConns      int           `config:"DB_MAX_OPEN_CONNS" default:"50"`
	DBMinConns          int           `config:"DB_MIN_CONNS" default:"5"`
	DBMaxIdleConns      int           `config:"DB_MAX_IDLE_CONNS" default:"25"`
	DBConnMaxLifetime   time.Duration `config:"DB_CONN_MAX_LIFETIME" default:"30m"`
	DBConnMaxIdleTime   time.Duration `config:"DB_CONN_MAX_IDLE_TIME" default:"5m"`
	DBHealthCheckPeriod time.Duration `config:"DB_HEALTH_CHECK_PERIOD" default:"1m"`
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
	poolSettings := di.poolSettings()
	cfg.MaxConns = int32(poolSettings.MaxOpenConns)
	cfg.MinConns = int32(poolSettings.MinConns)
	cfg.MaxConnIdleTime = poolSettings.ConnMaxIdleTime
	cfg.MaxConnLifetime = poolSettings.ConnMaxLifetime
	cfg.HealthCheckPeriod = poolSettings.HealthCheckPeriod

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
		otelsql.WithAttributesGetter(withQueryAttributes(di.Logger)),
		otelsql.WithInstrumentAttributesGetter(withQueryAttributes(di.Logger)),
	)
	di.db.SetMaxOpenConns(poolSettings.MaxOpenConns)
	di.db.SetMaxIdleConns(poolSettings.MaxIdleConns)
	di.db.SetConnMaxLifetime(poolSettings.ConnMaxLifetime)
	di.db.SetConnMaxIdleTime(poolSettings.ConnMaxIdleTime)

	di.metricRegistration, err = otelsql.RegisterDBStatsMetrics(
		di.db,
		dbSystemAttributes,
	)
	if err != nil {
		return ctx, fmt.Errorf("failed to register db stats metrics: %w", err)
	}

	// Run migrations
	if !di.SkipMigration {
		if err := di.runMigrations(); err != nil {
			return ctx, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	depend.Register(di.db)

	return ctx, nil
}

// poolSettings resolves the DB pool settings, applying defaults and safe bounds.
func (di InitDB) poolSettings() dbPoolSettings {
	settings := dbPoolSettings{
		MaxOpenConns:      defaultDBMaxOpenConns,
		MinConns:          defaultDBMinConns,
		MaxIdleConns:      defaultDBMaxIdleConns,
		ConnMaxLifetime:   defaultDBConnMaxLifetime,
		ConnMaxIdleTime:   defaultDBConnMaxIdleTime,
		HealthCheckPeriod: defaultDBHealthCheckPeriod,
	}

	if di.DBMaxOpenConns > 0 {
		settings.MaxOpenConns = di.DBMaxOpenConns
	}
	if di.DBMinConns > 0 {
		settings.MinConns = di.DBMinConns
	}
	if di.DBMaxIdleConns > 0 {
		settings.MaxIdleConns = di.DBMaxIdleConns
	}
	if di.DBConnMaxLifetime > 0 {
		settings.ConnMaxLifetime = di.DBConnMaxLifetime
	}
	if di.DBConnMaxIdleTime > 0 {
		settings.ConnMaxIdleTime = di.DBConnMaxIdleTime
	}
	if di.DBHealthCheckPeriod > 0 {
		settings.HealthCheckPeriod = di.DBHealthCheckPeriod
	}

	settings.MinConns = min(settings.MinConns, settings.MaxOpenConns)
	settings.MaxIdleConns = min(settings.MaxIdleConns, settings.MaxOpenConns)

	return settings
}

// runMigrations applies database migrations from the embedded filesystem using golang-migrate.
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

	if err := m.Up(); err != nil {
		switch {
		case errors.Is(err, migrate.ErrNoChange):
			di.Logger.Println("InitDB: no new migrations to apply")
			return nil
		default:
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
	} else {
		di.Logger.Println("InitDB: migrations applied successfully")
	}

	return nil
}

// Close terminates the database connection and unregisters metrics, logging any errors encountered during shutdown.
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

// withQueryAttributes returns a function that extracts SQL operation and table information from queries for telemetry attributes.
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
