package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anti-gravity-bit/api-onboarding-lab/internal/httpapi"
	"github.com/anti-gravity-bit/api-onboarding-lab/internal/store"
	"github.com/anti-gravity-bit/api-onboarding-lab/webhooksig"
)

const (
	fixtureToken  = "demo-api-token-not-for-production-use-32b"
	fixtureSecret = "demo-webhook-secret-not-for-production-use"
	validOrder    = `{"sku":"DEMO-WIDGET","quantity":2,"customer_ref":"cust_demo_1001"}`
)

var fixedNow = time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)

type errorEnvelope struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
		Details   []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details,omitempty"`
	} `json:"error"`
}

func newAPI(t *testing.T) (http.Handler, *store.Memory) {
	t.Helper()
	s := store.New()
	h, err := httpapi.New(httpapi.Options{
		APIToken:         fixtureToken,
		WebhookSecret:    fixtureSecret,
		WebhookTolerance: 300 * time.Second,
		MaxBodyBytes:     65536,
		Store:            s,
		Now:              func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h, s
}

func request(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range headers {
		r.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	return w
}

func authHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + fixtureToken,
		"Content-Type":  "application/json",
	}
}

func decodeJSON[T any](t *testing.T, r io.Reader) T {
	t.Helper()
	var value T
	if err := json.NewDecoder(r).Decode(&value); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return value
}

func requireError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) errorEnvelope {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, status, w.Body.String())
	}
	envelope := decodeJSON[errorEnvelope](t, w.Body)
	if envelope.Error.Code != code {
		t.Fatalf("error code = %q, want %q", envelope.Error.Code, code)
	}
	if envelope.Error.RequestID == "" {
		t.Fatal("error request_id is empty")
	}
	return envelope
}

func createOrder(t *testing.T, h http.Handler) store.Order {
	t.Helper()
	w := request(t, h, http.MethodPost, "/v1/orders", validOrder, authHeaders())
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", w.Code, w.Body.String())
	}
	return decodeJSON[store.Order](t, w.Body)
}

func eventBody(id string) string {
	return fmt.Sprintf(`{"id":%q,"type":"order.accepted","order_id":"ord_0123456789abcdef","occurred_at":"2026-10-19T00:00:00Z"}`, id)
}

func webhookHeaders(body string, timestamp int64) map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Lab-Signature": webhooksig.Sign([]byte(fixtureSecret), timestamp, []byte(body)),
	}
}

func TestAcceptanceA1Health(t *testing.T) {
	h, _ := newAPI(t)
	w := request(t, h, http.MethodGet, "/healthz", "", nil)
	if w.Code != http.StatusOK || w.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("health response = %d %q", w.Code, w.Body.String())
	}
}

func TestAcceptanceA2A5Orders(t *testing.T) {
	h, _ := newAPI(t)

	order := createOrder(t, h)
	if !regexp.MustCompile(`^ord_[0-9a-f]{16}$`).MatchString(order.ID) {
		t.Errorf("A2 id = %q", order.ID)
	}
	if order.SKU != "DEMO-WIDGET" || order.Quantity != 2 || order.CustomerRef != "cust_demo_1001" || order.Status != "accepted" || !order.CreatedAt.Equal(fixedNow) {
		t.Errorf("A2 order = %+v", order)
	}

	w := request(t, h, http.MethodGet, "/v1/orders/"+order.ID, "", map[string]string{"Authorization": "Bearer " + fixtureToken})
	if w.Code != http.StatusOK {
		t.Fatalf("A3 status = %d; body=%s", w.Code, w.Body.String())
	}
	got := decodeJSON[store.Order](t, w.Body)
	if got != order {
		t.Errorf("A3 order = %+v, want %+v", got, order)
	}

	w = request(t, h, http.MethodGet, "/v1/orders/ord_ffffffffffffffff", "", map[string]string{"Authorization": "Bearer " + fixtureToken})
	requireError(t, w, http.StatusNotFound, "not_found")

	w = request(t, h, http.MethodGet, "/v1/orders/not-an-order", "", map[string]string{"Authorization": "Bearer " + fixtureToken})
	envelope := requireError(t, w, http.StatusBadRequest, "validation_error")
	if envelope.Error.Message != "request validation failed" || len(envelope.Error.Details) == 0 || envelope.Error.Details[0].Field != "id" {
		t.Errorf("A5 error = %+v", envelope.Error)
	}
}

func TestAcceptanceA6BearerAuthentication(t *testing.T) {
	h, _ := newAPI(t)
	for _, path := range []string{"/v1/orders", "/v1/orders/ord_0123456789abcdef"} {
		var message string
		for _, authorization := range []string{"", "Bearer wrong-token"} {
			headers := map[string]string{"Content-Type": "application/json"}
			if authorization != "" {
				headers["Authorization"] = authorization
			}
			method, body := http.MethodGet, ""
			if path == "/v1/orders" {
				method, body = http.MethodPost, validOrder
			}
			w := request(t, h, method, path, body, headers)
			envelope := requireError(t, w, http.StatusUnauthorized, "unauthorized")
			if message == "" {
				message = envelope.Error.Message
			} else if envelope.Error.Message != message {
				t.Errorf("%s missing and wrong token messages differ", path)
			}
			if strings.Contains(w.Body.String(), fixtureToken) || strings.Contains(w.Body.String(), "wrong-token") {
				t.Errorf("%s response leaked a token", path)
			}
		}
	}
}

func TestAcceptanceA7MalformedOrderJSON(t *testing.T) {
	h, _ := newAPI(t)
	w := request(t, h, http.MethodPost, "/v1/orders", `{"sku":`, authHeaders())
	requireError(t, w, http.StatusBadRequest, "bad_request")
}

func TestAcceptanceA8OrderValidation(t *testing.T) {
	tests := []struct {
		name, body, field string
	}{
		{"unknown field", `{"sku":"DEMO-WIDGET","quantity":2,"customer_ref":"cust_demo_1001","extra":true}`, "extra"},
		{"missing field", `{"sku":"DEMO-WIDGET","quantity":2}`, "customer_ref"},
		{"bad sku", `{"sku":"bad sku","quantity":2,"customer_ref":"cust_demo_1001"}`, "sku"},
		{"bad customer ref", `{"sku":"DEMO-WIDGET","quantity":2,"customer_ref":"bad ref"}`, "customer_ref"},
		{"quantity low", `{"sku":"DEMO-WIDGET","quantity":0,"customer_ref":"cust_demo_1001"}`, "quantity"},
		{"quantity high", `{"sku":"DEMO-WIDGET","quantity":101,"customer_ref":"cust_demo_1001"}`, "quantity"},
		{"quantity fraction", `{"sku":"DEMO-WIDGET","quantity":2.5,"customer_ref":"cust_demo_1001"}`, "quantity"},
		{"quantity string", `{"sku":"DEMO-WIDGET","quantity":"2","customer_ref":"cust_demo_1001"}`, "quantity"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newAPI(t)
			w := request(t, h, http.MethodPost, "/v1/orders", tt.body, authHeaders())
			envelope := requireError(t, w, http.StatusBadRequest, "validation_error")
			if envelope.Error.Message != "request validation failed" || len(envelope.Error.Details) == 0 || envelope.Error.Details[0].Field != tt.field {
				t.Fatalf("validation error = %+v", envelope.Error)
			}
			if tt.field == "quantity" && envelope.Error.Details[0].Message != "must be an integer from 1 to 100" {
				t.Errorf("quantity message = %q", envelope.Error.Details[0].Message)
			}
		})
	}
}

func TestAcceptanceA9PayloadLimit(t *testing.T) {
	h, _ := newAPI(t)
	large := strings.Repeat("x", 65537)
	w := request(t, h, http.MethodPost, "/v1/orders", large, authHeaders())
	requireError(t, w, http.StatusRequestEntityTooLarge, "payload_too_large")

	headers := webhookHeaders(large, fixedNow.Unix())
	w = request(t, h, http.MethodPost, "/v1/webhooks/order", large, headers)
	requireError(t, w, http.StatusRequestEntityTooLarge, "payload_too_large")
}

func TestAcceptanceA10ContentType(t *testing.T) {
	for _, path := range []string{"/v1/orders", "/v1/webhooks/order"} {
		t.Run(path, func(t *testing.T) {
			h, _ := newAPI(t)
			body := validOrder
			headers := authHeaders()
			if path == "/v1/webhooks/order" {
				body = eventBody("evt_content_type")
				headers = webhookHeaders(body, fixedNow.Unix())
			}
			headers["Content-Type"] = "text/plain"
			w := request(t, h, http.MethodPost, path, body, headers)
			requireError(t, w, http.StatusUnsupportedMediaType, "unsupported_media_type")

			headers["Content-Type"] = "application/json; charset=utf-8"
			w = request(t, h, http.MethodPost, path, body, headers)
			if w.Code != http.StatusCreated && w.Code != http.StatusAccepted {
				t.Errorf("accepted charset status = %d; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAcceptanceA11ValidWebhook(t *testing.T) {
	h, _ := newAPI(t)
	body := eventBody("evt_demo_1001")
	w := request(t, h, http.MethodPost, "/v1/webhooks/order", body, webhookHeaders(body, fixedNow.Unix()))
	if w.Code != http.StatusAccepted || w.Body.String() != "{\"status\":\"accepted\",\"duplicate\":false,\"event_id\":\"evt_demo_1001\"}\n" {
		t.Fatalf("webhook response = %d %q", w.Code, w.Body.String())
	}
}

func TestAcceptanceA12BadWebhookSignaturesAreNotStored(t *testing.T) {
	cases := []struct {
		name, signature string
	}{
		{"missing", ""},
		{"malformed", "t=abc,v1=xyz"},
		{"mismatch", webhooksig.Sign([]byte(fixtureSecret), fixedNow.Unix(), []byte("different"))},
	}
	for i, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newAPI(t)
			id := fmt.Sprintf("evt_bad_sig_%d", i)
			body := eventBody(id)
			headers := map[string]string{"Content-Type": "application/json"}
			if tt.signature != "" {
				headers["Lab-Signature"] = tt.signature
			}
			w := request(t, h, http.MethodPost, "/v1/webhooks/order", body, headers)
			requireError(t, w, http.StatusUnauthorized, "unauthorized")
			if strings.Contains(w.Body.String(), fixtureSecret) || strings.Contains(w.Body.String(), body) {
				t.Fatal("signature error leaked a secret or request body")
			}
			w = request(t, h, http.MethodPost, "/v1/webhooks/order", body, webhookHeaders(body, fixedNow.Unix()))
			if w.Code != http.StatusAccepted {
				t.Fatalf("event was stored after failed signature: status %d", w.Code)
			}
		})
	}
}

func TestAcceptanceA13WebhookTimestampWindow(t *testing.T) {
	for _, skew := range []int64{-301, 301} {
		h, _ := newAPI(t)
		id := fmt.Sprintf("evt_skew_%d", skew+400)
		body := eventBody(id)
		w := request(t, h, http.MethodPost, "/v1/webhooks/order", body, webhookHeaders(body, fixedNow.Unix()+skew))
		requireError(t, w, http.StatusUnauthorized, "unauthorized")
		w = request(t, h, http.MethodPost, "/v1/webhooks/order", body, webhookHeaders(body, fixedNow.Unix()))
		if w.Code != http.StatusAccepted {
			t.Errorf("skew %d stored event after rejection: %d", skew, w.Code)
		}
	}
	for _, skew := range []int64{-300, 300} {
		h, _ := newAPI(t)
		body := eventBody(fmt.Sprintf("evt_boundary_%d", skew+400))
		w := request(t, h, http.MethodPost, "/v1/webhooks/order", body, webhookHeaders(body, fixedNow.Unix()+skew))
		if w.Code != http.StatusAccepted {
			t.Errorf("boundary skew %d status = %d; body=%s", skew, w.Code, w.Body.String())
		}
	}
}

func TestAcceptanceA14DuplicateWebhook(t *testing.T) {
	h, _ := newAPI(t)
	body := eventBody("evt_duplicate")
	headers := webhookHeaders(body, fixedNow.Unix())
	if got := request(t, h, http.MethodPost, "/v1/webhooks/order", body, headers).Code; got != http.StatusAccepted {
		t.Fatalf("first status = %d", got)
	}
	w := request(t, h, http.MethodPost, "/v1/webhooks/order", body, headers)
	if w.Code != http.StatusOK || w.Body.String() != "{\"status\":\"duplicate\",\"duplicate\":true,\"event_id\":\"evt_duplicate\"}\n" {
		t.Fatalf("duplicate response = %d %q", w.Code, w.Body.String())
	}
}

func TestAcceptanceA15ConcurrentDuplicateWebhook(t *testing.T) {
	h, s := newAPI(t)
	body := eventBody("evt_concurrent")
	headers := webhookHeaders(body, fixedNow.Unix())
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			r := httptest.NewRequest(http.MethodPost, "/v1/webhooks/order", strings.NewReader(body))
			for key, value := range headers {
				r.Header.Set(key, value)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			statuses <- w.Code
		}()
	}
	close(start)
	wg.Wait()
	close(statuses)
	got := make([]int, 0, 2)
	for status := range statuses {
		got = append(got, status)
	}
	sort.Ints(got)
	if len(got) != 2 || got[0] != http.StatusOK || got[1] != http.StatusAccepted {
		t.Fatalf("statuses = %v, want [200 202]", got)
	}
	if s.EventCount() != 1 {
		t.Fatalf("stored event count = %d, want 1", s.EventCount())
	}
}

func TestAcceptanceA16InvalidWebhookBodyIsNotStored(t *testing.T) {
	t.Run("malformed JSON", func(t *testing.T) {
		h, _ := newAPI(t)
		body := `{"id":`
		w := request(t, h, http.MethodPost, "/v1/webhooks/order", body, webhookHeaders(body, fixedNow.Unix()))
		requireError(t, w, http.StatusBadRequest, "bad_request")
	})

	t.Run("invalid schema", func(t *testing.T) {
		h, _ := newAPI(t)
		bad := `{"id":"evt_schema","type":"wrong","order_id":"ord_0123456789abcdef","occurred_at":"2026-10-19T00:00:00Z"}`
		w := request(t, h, http.MethodPost, "/v1/webhooks/order", bad, webhookHeaders(bad, fixedNow.Unix()))
		requireError(t, w, http.StatusBadRequest, "validation_error")
		good := eventBody("evt_schema")
		w = request(t, h, http.MethodPost, "/v1/webhooks/order", good, webhookHeaders(good, fixedNow.Unix()))
		if w.Code != http.StatusAccepted {
			t.Fatalf("invalid schema stored event: status %d", w.Code)
		}
	})
}

func TestAcceptanceA17RequestID(t *testing.T) {
	h, _ := newAPI(t)
	w := request(t, h, http.MethodGet, "/missing", "", map[string]string{"X-Request-Id": "client.request-_1"})
	envelope := requireError(t, w, http.StatusNotFound, "not_found")
	if w.Header().Get("X-Request-Id") != "client.request-_1" || envelope.Error.RequestID != "client.request-_1" {
		t.Fatalf("safe request id was not echoed: header=%q body=%q", w.Header().Get("X-Request-Id"), envelope.Error.RequestID)
	}

	for _, unsafe := range []string{"", "contains space", strings.Repeat("a", 65), "slash/invalid"} {
		w = request(t, h, http.MethodGet, "/missing", "", map[string]string{"X-Request-Id": unsafe})
		envelope = requireError(t, w, http.StatusNotFound, "not_found")
		generated := w.Header().Get("X-Request-Id")
		if !regexp.MustCompile(`^req_[0-9a-f]{16}$`).MatchString(generated) || envelope.Error.RequestID != generated {
			t.Errorf("unsafe %q generated header=%q body=%q", unsafe, generated, envelope.Error.RequestID)
		}
	}
}

func TestAcceptanceA18NewStoreLosesState(t *testing.T) {
	h1, _ := newAPI(t)
	order := createOrder(t, h1)
	body := eventBody("evt_restart")
	if got := request(t, h1, http.MethodPost, "/v1/webhooks/order", body, webhookHeaders(body, fixedNow.Unix())).Code; got != http.StatusAccepted {
		t.Fatalf("first event status = %d", got)
	}

	h2, _ := newAPI(t)
	w := request(t, h2, http.MethodGet, "/v1/orders/"+order.ID, "", map[string]string{"Authorization": "Bearer " + fixtureToken})
	requireError(t, w, http.StatusNotFound, "not_found")
	w = request(t, h2, http.MethodPost, "/v1/webhooks/order", body, webhookHeaders(body, fixedNow.Unix()))
	if w.Code != http.StatusAccepted {
		t.Fatalf("new store retained event id: status %d", w.Code)
	}
}

func TestAcceptanceA19GracefulShutdown(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		once.Do(func() { close(started) })
		<-release
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "{\"status\":\"ok\"}\n")
	})
	srv := httpapi.NewServer("127.0.0.1:0", h)
	if srv.ReadHeaderTimeout != 5*time.Second || srv.ReadTimeout != 10*time.Second || srv.WriteTimeout != 15*time.Second || srv.IdleTimeout != 60*time.Second {
		t.Fatalf("server timeouts = %s/%s/%s/%s", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.WriteTimeout, srv.IdleTimeout)
	}
	listener := newMemoryListener()
	serveDone := make(chan error, 1)
	go func() { serveDone <- srv.Serve(listener) }()

	transport := &http.Transport{DialContext: listener.DialContext}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	responseDone := make(chan error, 1)
	go func() {
		resp, err := client.Get("http://memory.test/")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			err = resp.Body.Close()
		}
		responseDone <- err
	}()
	<-started

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- srv.Shutdown(t.Context()) }()
	select {
	case <-listener.closed:
	case <-time.After(time.Second):
		t.Fatal("Shutdown did not close the listener")
	}
	if resp, err := client.Get("http://memory.test/"); err == nil {
		_ = resp.Body.Close()
		t.Fatal("server accepted a new request during shutdown")
	}
	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned before in-flight request completed: %v", err)
	default:
	}
	close(release)
	if err := <-responseDone; err != nil {
		t.Fatalf("in-flight request: %v", err)
	}
	if err := <-shutdownDone; err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := <-serveDone; err != http.ErrServerClosed {
		t.Fatalf("Serve = %v, want http.ErrServerClosed", err)
	}
}

type memoryListener struct {
	connections chan net.Conn
	closed      chan struct{}
	closeOnce   sync.Once
}

func newMemoryListener() *memoryListener {
	return &memoryListener{
		connections: make(chan net.Conn),
		closed:      make(chan struct{}),
	}
}

func (l *memoryListener) Accept() (net.Conn, error) {
	select {
	case connection := <-l.connections:
		return connection, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *memoryListener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

func (l *memoryListener) Addr() net.Addr { return memoryAddr("memory") }

func (l *memoryListener) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	select {
	case <-l.closed:
		return nil, net.ErrClosed
	default:
	}
	client, server := net.Pipe()
	select {
	case l.connections <- server:
		return client, nil
	case <-l.closed:
		_ = client.Close()
		_ = server.Close()
		return nil, net.ErrClosed
	case <-ctx.Done():
		_ = client.Close()
		_ = server.Close()
		return nil, ctx.Err()
	}
}

type memoryAddr string

func (a memoryAddr) Network() string { return string(a) }
func (a memoryAddr) String() string  { return string(a) }

func TestUnmatchedAndWrongMethodErrors(t *testing.T) {
	h, _ := newAPI(t)
	requireError(t, request(t, h, http.MethodGet, "/not-a-route", "", nil), http.StatusNotFound, "not_found")
	requireError(t, request(t, h, http.MethodPost, "/healthz", "", nil), http.StatusMethodNotAllowed, "method_not_allowed")
}

func TestResponsesDoNotContainRequestBodies(t *testing.T) {
	h, _ := newAPI(t)
	marker := "sensitive-body-marker"
	body := `{"sku":"` + marker + `"}`
	w := request(t, h, http.MethodPost, "/v1/orders", body, authHeaders())
	if bytes.Contains(w.Body.Bytes(), []byte(marker)) {
		t.Fatal("error response contains request body content")
	}
}
