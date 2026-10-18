package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/anti-gravity-bit/api-onboarding-lab/internal/config"
)

const (
	fixtureToken  = "demo-api-token-not-for-production-use-32b"
	fixtureSecret = "demo-webhook-secret-not-for-production-use"
)

func lookup(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestDefaults(t *testing.T) {
	cfg, err := config.FromLookup(lookup(map[string]string{
		"API_TOKEN":      fixtureToken,
		"WEBHOOK_SECRET": fixtureSecret,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "127.0.0.1:8080" || cfg.WebhookTolerance != 300*time.Second || cfg.MaxBodyBytes != 65536 || cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("defaults = %+v", cfg)
	}
}

func TestOverrides(t *testing.T) {
	cfg, err := config.FromLookup(lookup(map[string]string{
		"API_TOKEN":                 fixtureToken,
		"WEBHOOK_SECRET":            fixtureSecret,
		"ADDR":                      ":9000",
		"WEBHOOK_TOLERANCE_SECONDS": "42",
		"MAX_BODY_BYTES":            "2048",
		"SHUTDOWN_TIMEOUT":          "750ms",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9000" || cfg.WebhookTolerance != 42*time.Second || cfg.MaxBodyBytes != 2048 || cfg.ShutdownTimeout != 750*time.Millisecond {
		t.Fatalf("overrides = %+v", cfg)
	}
}

func TestAcceptanceA20RequiredSecrets(t *testing.T) {
	secretValues := []string{"", "short", "sensitive-but-short"}
	tests := []struct {
		name, key string
	}{
		{"API token", "API_TOKEN"},
		{"webhook secret", "WEBHOOK_SECRET"},
	}
	for _, tt := range tests {
		for _, value := range secretValues {
			t.Run(tt.name+"/length-"+string(rune(len(value))), func(t *testing.T) {
				values := map[string]string{"API_TOKEN": fixtureToken, "WEBHOOK_SECRET": fixtureSecret}
				values[tt.key] = value
				_, err := config.FromLookup(lookup(values))
				if err == nil {
					t.Fatal("expected startup configuration error")
				}
				if value != "" && strings.Contains(err.Error(), value) {
					t.Fatal("configuration error printed credential value")
				}
			})
		}
	}
}
