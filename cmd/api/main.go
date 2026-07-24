package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/toanle88/Tally/internal/platform/httpx"
)

const (
	defaultHTTPAddress = ":8080"
	shutdownTimeout    = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Printf("API stopped with error: %v", err)
		os.Exit(1)
	}

}

func run() error {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = defaultHTTPAddress
	}

	router := chi.NewRouter()
	router.Get("/health/live", httpx.Liveness)

	server := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	listenError := make(chan error, 1)

	go func() {
		log.Printf("API listening on %s", address)
		listenError <- server.ListenAndServe()
	}()

	select {
	case err := <-listenError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("listen and serve: %w", err)
	case <-shutdownSignal.Done():
		log.Printf("API shutting down with timeout %s", shutdownTimeout)
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	err := <-listenError
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server stopped: %w", err)
	}

	log.Printf("API stopped gracefully")
	return nil
}
