package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/thomazDeveloper/go-rest-api-boilerplate/internal/config"
)

// customLogger wraps the default logger to ignore ErrRecordNotFound
type customLogger struct {
	logger.Interface
}

func (l customLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	// Don't log "record not found" errors as they are expected in many cases
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return
	}
	l.Interface.Trace(ctx, begin, fc, err)
}

func (l customLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	// Don't log "record not found" errors as they are expected in many cases
	if len(data) > 0 {
		if err, ok := data[0].(error); ok && errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
	}
	l.Interface.Error(ctx, msg, data...)
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg Config) (*bun.DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
		config, err := pgx.ParseConfig(dsn)
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
if err != nil {
	return nil, fmt.Errorf("failed to connect to database: %w", err)
}

sqldb := stdlib.OpenDBFromPool(pool)
db := bun.NewDB(sqldb, pgdialect.New())
	log.Println("Database connection established")
	return db, nil
}

// NewPostgresDBFromDatabaseConfig creates a new PostgreSQL DB connection from typed config
func NewPostgresDBFromDatabaseConfig(cfg config.DatabaseConfig) (*bun.DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
		config, err := pgx.ParseConfig(dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: customLogger{logger.Default.LogMode(logger.Info)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres database: %w", err)
	}

	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

sqldb := stdlib.OpenDB(*config)
db := bun.NewDB(sqldb, pgdialect.New())

	return db, nil
}

// NewSQLiteDB creates a new SQLite database connection (for testing)
func NewSQLiteDB(dbPath string) (*bun.DB, error) {

sqldb, err := sql.Open(sqliteshim.ShimName, "file:test.db?cache=shared&mode=rwc")
if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite database: %w", err)
	}
db := bun.NewDB(sqldb, sqlitedialect.New())

	return db, nil
}

// LoadConfigFromEnv loads database configuration using Viper (env overrides + defaults)
func LoadConfigFromEnv() Config {
	return Config{
		Host:     viper.GetString("database.host"),
		Port:     viper.GetInt("database.port"),
		User:     viper.GetString("database.user"),
		Password: viper.GetString("database.password"),
		Name:     viper.GetString("database.name"),
		SSLMode:  viper.GetString("database.sslmode"),
	}
}
