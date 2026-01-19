package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// connect initializes a connection pool to postgres
func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	// fetching url from env or building from individual vars
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		// build from individual environment variables
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "user")
		password := getEnv("DB_PASSWORD", "password")
		dbname := getEnv("DB_NAME", "location_db")
		sslmode := getEnv("DB_SSLMODE", "disable")

		dbUrl = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			user, password, host, port, dbname, sslmode)
	}

	config, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse db config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// simple ping to verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping db: %w", err)
	}

	return pool, nil
}

// getEnv gets an env vars or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
