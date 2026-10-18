package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const (
	fixtureToken  = "demo-api-token-not-for-production-use-32b"
	fixtureSecret = "demo-webhook-secret-not-for-production-use"
)

func TestAcceptanceA20StartupFailure(t *testing.T) {
	tests := []struct {
		name          string
		apiToken      string
		webhookSecret string
	}{
		{"missing API token", "", fixtureSecret},
		{"short API token", "sensitive-short-token", fixtureSecret},
		{"missing webhook secret", fixtureToken, ""},
		{"short webhook secret", fixtureToken, "sensitive-short-secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestLabdStartupHelper$")
			cmd.Env = scrubEnvironment(os.Environ(), "API_TOKEN", "WEBHOOK_SECRET", "LABD_STARTUP_HELPER")
			cmd.Env = append(cmd.Env,
				"LABD_STARTUP_HELPER=1",
				"API_TOKEN="+tt.apiToken,
				"WEBHOOK_SECRET="+tt.webhookSecret,
			)
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("startup succeeded with invalid credentials")
			}
			if _, ok := err.(*exec.ExitError); !ok {
				t.Fatalf("startup error = %T %v", err, err)
			}
			for _, secret := range []string{tt.apiToken, tt.webhookSecret} {
				if secret != "" && strings.Contains(string(output), secret) {
					t.Fatal("startup output contained a credential value")
				}
			}
		})
	}
}

func TestLabdStartupHelper(t *testing.T) {
	if os.Getenv("LABD_STARTUP_HELPER") != "1" {
		return
	}
	if err := run(context.Background(), os.Getenv); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(23)
	}
	os.Exit(0)
}

func scrubEnvironment(environment []string, keys ...string) []string {
	prefixes := make([]string, 0, len(keys))
	for _, key := range keys {
		prefixes = append(prefixes, key+"=")
	}
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		keep := true
		for _, prefix := range prefixes {
			if strings.HasPrefix(entry, prefix) {
				keep = false
				break
			}
		}
		if keep {
			result = append(result, entry)
		}
	}
	return result
}
