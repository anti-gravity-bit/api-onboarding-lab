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

	"github.com/anti-gravity-bit/api-onboarding-lab/internal/config"
	"github.com/anti-gravity-bit/api-onboarding-lab/internal/httpapi"
	"github.com/anti-gravity-bit/api-onboarding-lab/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Getenv); err != nil {
		log.New(os.Stderr, "labd: ", 0).Print(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, lookup func(string) string) error {
	cfg, err := config.FromLookup(lookup)
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	handler, err := httpapi.New(httpapi.Options{
		APIToken:         cfg.APIToken,
		WebhookSecret:    cfg.WebhookSecret,
		WebhookTolerance: cfg.WebhookTolerance,
		MaxBodyBytes:     cfg.MaxBodyBytes,
		Store:            store.New(),
	})
	if err != nil {
		return fmt.Errorf("initialize HTTP API: %w", err)
	}
	server := httpapi.NewServer(cfg.Addr, handler)
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.ListenAndServe()
	}()

	select {
	case err := <-serveResult:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		err := <-serveResult
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}
}
