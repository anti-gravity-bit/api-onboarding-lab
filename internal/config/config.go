package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const minimumCredentialBytes = 32

type Config struct {
	Addr             string
	APIToken         string
	WebhookSecret    string
	WebhookTolerance time.Duration
	MaxBodyBytes     int64
	ShutdownTimeout  time.Duration
}

func Load() (Config, error) {
	return FromLookup(os.Getenv)
}

func FromLookup(lookup func(string) string) (Config, error) {
	if lookup == nil {
		return Config{}, errors.New("environment lookup is required")
	}
	cfg := Config{
		Addr:             valueOrDefault(lookup("ADDR"), "127.0.0.1:8080"),
		APIToken:         lookup("API_TOKEN"),
		WebhookSecret:    lookup("WEBHOOK_SECRET"),
		WebhookTolerance: 300 * time.Second,
		MaxBodyBytes:     65536,
		ShutdownTimeout:  10 * time.Second,
	}
	if len([]byte(cfg.APIToken)) < minimumCredentialBytes {
		return Config{}, errors.New("API_TOKEN must be at least 32 bytes")
	}
	if len([]byte(cfg.WebhookSecret)) < minimumCredentialBytes {
		return Config{}, errors.New("WEBHOOK_SECRET must be at least 32 bytes")
	}

	if raw := lookup("WEBHOOK_TOLERANCE_SECONDS"); raw != "" {
		seconds, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || seconds < 0 || seconds > int64(^uint64(0)>>1)/int64(time.Second) {
			return Config{}, errors.New("WEBHOOK_TOLERANCE_SECONDS must be a non-negative integer")
		}
		cfg.WebhookTolerance = time.Duration(seconds) * time.Second
	}
	if raw := lookup("MAX_BODY_BYTES"); raw != "" {
		maximum, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || maximum < 1 {
			return Config{}, errors.New("MAX_BODY_BYTES must be a positive integer")
		}
		cfg.MaxBodyBytes = maximum
	}
	if raw := lookup("SHUTDOWN_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return Config{}, errors.New("SHUTDOWN_TIMEOUT must be a positive Go duration")
		}
		cfg.ShutdownTimeout = timeout
	}
	if cfg.Addr == "" {
		return Config{}, fmt.Errorf("ADDR must not be empty")
	}
	return cfg, nil
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
