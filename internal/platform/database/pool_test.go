package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOpenRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	password := generateTestPassword(t)

	databaseURL := fmt.Sprintf(
		"postgres://test_user:%s@127.0.0.1:1/tally_test?sslmode=disable",
		url.QueryEscape(password),
	)

	valid := Config{
		DatabaseURL:    databaseURL,
		MaxConnections: 4,
		ConnectTimeout: time.Second,
	}

	tests := []struct {
		name   string
		ctx    context.Context
		mutate func(*Config)
	}{
		{
			name: "nil context",
			ctx:  nil,
		},
		{
			name: "empty database URL",
			ctx:  context.Background(),
			mutate: func(cfg *Config) {
				cfg.DatabaseURL = "   "
			},
		},
		{
			name: "malformed database URL",
			ctx:  context.Background(),
			mutate: func(cfg *Config) {
				cfg.DatabaseURL = "postgres://tally:secret@%zz/tally"
			},
		},
		{
			name: "below minimum connections",
			ctx:  context.Background(),
			mutate: func(cfg *Config) {
				cfg.MaxConnections = 1
			},
		},
		{
			name: "above maximum connections",
			ctx:  context.Background(),
			mutate: func(cfg *Config) {
				cfg.MaxConnections = 201
			},
		},
		{
			name: "zero connect timeout",
			ctx:  context.Background(),
			mutate: func(cfg *Config) {
				cfg.ConnectTimeout = 0
			},
		},
		{
			name: "negative connect timeout",
			ctx:  context.Background(),
			mutate: func(cfg *Config) {
				cfg.ConnectTimeout = -time.Second
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}

			pool, err := Open(tt.ctx, cfg)
			if pool != nil {
				pool.Close()
				t.Fatal("Open returned a pool for invalid configuration")
			}
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("Open error = %v, want ErrInvalidConfig", err)
			}
			assertErrorExcludes(t, err, cfg.DatabaseURL, "secret", "local-only")
		})
	}
}

func TestOpenReportsConnectionRefusalWithoutExposingCredentials(t *testing.T) {
	t.Parallel()

	address := closedTCPAddress(t)
	const password = "connection-refusal-secret"
	databaseURL := postgresURL(address, password)

	pool, err := Open(context.Background(), Config{
		DatabaseURL:    databaseURL,
		MaxConnections: 2,
		ConnectTimeout: time.Second,
	})
	if pool != nil {
		pool.Close()
		t.Fatal("Open returned a pool for an unavailable database")
	}
	if !errors.Is(err, ErrConnectionFailed) {
		t.Fatalf("Open error = %v, want ErrConnectionFailed", err)
	}
	assertErrorExcludes(t, err, password, databaseURL)
}

func TestOpenPreservesCanceledContextWithoutExposingCredentials(t *testing.T) {
	t.Parallel()

	const password = "canceled-context-secret"
	databaseURL := postgresURL("127.0.0.1:5432", password)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	pool, err := Open(ctx, Config{
		DatabaseURL:    databaseURL,
		MaxConnections: 2,
		ConnectTimeout: time.Second,
	})
	if pool != nil {
		pool.Close()
		t.Fatal("Open returned a pool for an already-canceled context")
	}
	if !errors.Is(err, ErrConnectionFailed) {
		t.Fatalf("Open error = %v, want ErrConnectionFailed", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Open error = %v, want context.Canceled", err)
	}
	assertErrorExcludes(t, err, password, databaseURL)
}

func TestOpenBoundsConnectivityCheckByTimeout(t *testing.T) {
	address := stallingTCPAddress(t)
	const password = "connect-timeout-secret"
	databaseURL := postgresURL(address, password)

	pool, err := Open(context.Background(), Config{
		DatabaseURL:    databaseURL,
		MaxConnections: 2,
		ConnectTimeout: 200 * time.Millisecond,
	})
	if pool != nil {
		pool.Close()
		t.Fatal("Open returned a pool when the PostgreSQL handshake never completed")
	}
	if !errors.Is(err, ErrConnectionFailed) {
		t.Fatalf("Open error = %v, want ErrConnectionFailed", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Open error = %v, want context.DeadlineExceeded", err)
	}
	assertErrorExcludes(t, err, password, databaseURL)
}

func closedTCPAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve local TCP address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release local TCP address: %v", err)
	}
	return address
}

func stallingTCPAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for stalled PostgreSQL handshake: %v", err)
	}

	stop := make(chan struct{})
	t.Cleanup(func() {
		close(stop)
		_ = listener.Close()
	})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				<-stop
			}(conn)
		}
	}()

	return listener.Addr().String()
}

func postgresURL(address, password string) string {
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword("tally", password),
		Host:     address,
		Path:     "/tally",
		RawQuery: "sslmode=disable",
	}).String()
}

func assertErrorExcludes(t *testing.T, err error, forbidden ...string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error")
	}
	for _, value := range forbidden {
		if value != "" && strings.Contains(err.Error(), value) {
			t.Fatalf("error exposed forbidden connection detail %q: %v", value, err)
		}
	}
}

func generateTestPassword(t *testing.T) string {
	t.Helper()

	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatalf("generate test password: %v", err)
	}

	return hex.EncodeToString(value[:])
}
