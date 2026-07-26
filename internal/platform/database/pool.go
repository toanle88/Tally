package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	minPoolConnections int32 = 2
	maxPoolConnections int32 = 200
)

var (
	ErrInvalidConfig    = errors.New("invalid database configuration")
	ErrConnectionFailed = errors.New("database connection failed")
)

type Config struct {
	DatabaseURL    string
	MaxConnections int32
	ConnectTimeout time.Duration
}

func Open(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if err := validate(ctx, cfg); err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, invalidConfigError("database URL is required")
	}
	poolConfig.MaxConns = cfg.MaxConnections

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, connectionError(ctx)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, connectionError(pingCtx)
	}

	return pool, nil
}

func validate(ctx context.Context, cfg Config) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", ErrInvalidConfig)
	}

	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return invalidConfigError("database URL is required")
	}

	if cfg.MaxConnections < minPoolConnections || cfg.MaxConnections > maxPoolConnections {
		return invalidConfigError("maximum connections must be between 2 and 200")
	}

	if cfg.ConnectTimeout <= 0 {
		return invalidConfigError("connection timeout must be greater than zero")
	}

	return nil
}

func invalidConfigError(s string) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfig, s)
}

func connectionError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return errors.Join(ErrConnectionFailed, err)
	}
	return ErrConnectionFailed
}
