package config

import (
	"context"
	"fmt"
	"time"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
	"set-and-trend/backend/internal/db"
)

func NewDatabase(ctx context.Context, cfg *Config) (*db.Queries, *pgxpool.Pool, error) {

	//creat Url PostgreSQL
	dsn := fmt.Sprintf(
	"postgres://%s:%s@%s:%s/%s?sslmode=%s",
	cfg.DBUser,
	url.QueryEscape(cfg.DBPassword),
	cfg.DBHost,
	cfg.DBPort,
	cfg.DBName,
	cfg.DBSSLMode,
	)
	//convert credentials in tu URL and return  config obj
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("pgxpool.ParseConfig: %w", err)
	}
	//define rules for connection pool
	poolConfig.MaxConns = 100
	poolConfig.MinConns = 10
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second
	
	//create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("pgxpool.NewWithConfig: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("ping: %w", err)
	}

	return db.New(pool), pool, nil
}
