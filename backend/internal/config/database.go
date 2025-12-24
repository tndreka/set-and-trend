package config

import (
	"context"
	"fmt"
	
	"github.com/jackc/pgx/v5/pgxpool"
	"set-and-trend/backend/internal/db"
)

func NewDatabase(ctx context.Context, cfg *Config) (*db.Queries, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	
	return db.New(pool), nil
}
