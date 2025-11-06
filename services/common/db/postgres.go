package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool은 Postgres 연결 풀을 생성합니다.
func NewPool(ctx context.Context, uri string) (*pgxpool.Pool, error) {
	if uri == "" {
		return nil, errors.New("postgres uri is empty")
	}

	cfg, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return nil, fmt.Errorf("parse postgres uri: %w", err)
	}
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new postgres pool: %w", err)
	}

	return pool, nil
}

// Ping은 데이터베이스 연결을 확인합니다.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("nil postgres pool")
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}
	return nil
}
