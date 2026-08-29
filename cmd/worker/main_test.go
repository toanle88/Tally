package main

import (
	"context"
	"errors"
	"testing"
)

func TestRunFailsBeforeDatabaseSetupWithoutConsumers(t *testing.T) {
	if err := run(context.Background()); !errors.Is(err, ErrNoRegisteredConsumers) {
		t.Fatalf("run error = %v, want ErrNoRegisteredConsumers", err)
	}
}
