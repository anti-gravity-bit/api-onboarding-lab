package webhooksig_test

import (
	"testing"
	"time"

	"github.com/anti-gravity-bit/api-onboarding-lab/webhooksig"
)

func TestSignAndVerifyRawBody(t *testing.T) {
	secret := []byte("demo-webhook-secret-not-for-production-use")
	now := time.Unix(1710000000, 0)
	body := []byte("{ \"exact\": \"raw bytes\" }\n")
	signature := webhooksig.Sign(secret, now.Unix(), body)
	if err := webhooksig.Verify(secret, signature, body, now, 300*time.Second); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if err := webhooksig.Verify(secret, signature, []byte("{\"exact\":\"raw bytes\"}"), now, 300*time.Second); err == nil {
		t.Fatal("Verify accepted a normalized body")
	}
}

func TestVerifyRejectsMalformedHeaders(t *testing.T) {
	secret := []byte("demo-webhook-secret-not-for-production-use")
	now := time.Unix(1710000000, 0)
	for _, header := range []string{
		"", "t=1710000000, v1=" + string(make([]byte, 64)), "v1=00,t=1710000000",
		"t=1710000000,v1=ABCDEF", "t=1710000000,v1=zzzz", "t=1710000000,v1=00",
	} {
		if err := webhooksig.Verify(secret, header, nil, now, 300*time.Second); err == nil {
			t.Errorf("Verify accepted malformed header %q", header)
		}
	}
}
