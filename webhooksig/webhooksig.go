// Package webhooksig signs and verifies API Onboarding Lab webhook payloads.
package webhooksig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidSignature = errors.New("invalid webhook signature")

// Sign returns a Lab-Signature value for the timestamp and exact body bytes.
func Sign(secret []byte, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = fmt.Fprintf(mac, "%d.", timestamp)
	_, _ = mac.Write(body)
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}

// Verify checks the header shape, inclusive timestamp window, and HMAC over
// the timestamp, one period, and the unmodified request body.
func Verify(secret []byte, header string, body []byte, now time.Time, tolerance time.Duration) error {
	if tolerance < 0 {
		return ErrInvalidSignature
	}
	parts := strings.Split(header, ",")
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "t=") || !strings.HasPrefix(parts[1], "v1=") {
		return ErrInvalidSignature
	}
	timestampText := strings.TrimPrefix(parts[0], "t=")
	macText := strings.TrimPrefix(parts[1], "v1=")
	if timestampText == "" || len(macText) != sha256.Size*2 || macText != strings.ToLower(macText) {
		return ErrInvalidSignature
	}
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil {
		return ErrInvalidSignature
	}
	provided, err := hex.DecodeString(macText)
	if err != nil || len(provided) != sha256.Size {
		return ErrInvalidSignature
	}

	nowUnix := now.Unix()
	var difference uint64
	if nowUnix >= timestamp {
		difference = uint64(nowUnix) - uint64(timestamp)
	} else {
		difference = uint64(timestamp) - uint64(nowUnix)
	}
	if difference > uint64(tolerance/time.Second) {
		return ErrInvalidSignature
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = fmt.Fprintf(mac, "%d.", timestamp)
	_, _ = mac.Write(body)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return ErrInvalidSignature
	}
	return nil
}
