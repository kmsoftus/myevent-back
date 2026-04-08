package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

func Connect(ctx context.Context, databaseURL string, pool PoolConfig) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	if pool.MaxConns > 0 {
		cfg.MaxConns = pool.MaxConns
	}
	if pool.MinConns > 0 {
		cfg.MinConns = pool.MinConns
	}
	if pool.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = pool.MaxConnLifetime
	}
	if pool.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = pool.MaxConnIdleTime
	}

	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := p.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return p, nil
}
