package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/anti-gravity-bit/api-onboarding-lab/webhooksig"
)

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: signature TIMESTAMP")
		os.Exit(2)
	}
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		_, _ = fmt.Fprintln(os.Stderr, "WEBHOOK_SECRET is required")
		os.Exit(2)
	}
	timestamp, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "timestamp must be a base-10 Unix timestamp")
		os.Exit(2)
	}
	body, err := io.ReadAll(io.LimitReader(os.Stdin, 65537))
	if err != nil || len(body) > 65536 {
		_, _ = fmt.Fprintln(os.Stderr, "could not read a body of at most 65536 bytes")
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout, webhooksig.Sign([]byte(secret), timestamp, body))
}
