package httpapi

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/anti-gravity-bit/api-onboarding-lab/internal/store"
	"github.com/anti-gravity-bit/api-onboarding-lab/webhooksig"
)

var (
	skuPattern         = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{0,63}$`)
	customerRefPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	orderIDPattern     = regexp.MustCompile(`^ord_[0-9a-f]{16}$`)
	eventIDPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)
	requestIDPattern   = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
)

type Options struct {
	APIToken         string
	WebhookSecret    string
	WebhookTolerance time.Duration
	MaxBodyBytes     int64
	Store            *store.Memory
	Now              func() time.Time
}

type API struct {
	apiToken         string
	webhookSecret    []byte
	webhookTolerance time.Duration
	maxBodyBytes     int64
	store            *store.Memory
	now              func() time.Time
}

type fieldDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type errorBody struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	RequestID string        `json:"request_id"`
	Details   []fieldDetail `json:"details,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type webhookResult struct {
	Status    string `json:"status"`
	Duplicate bool   `json:"duplicate"`
	EventID   string `json:"event_id"`
}

func New(options Options) (*API, error) {
	if options.APIToken == "" {
		return nil, errors.New("API token is required")
	}
	if options.WebhookSecret == "" {
		return nil, errors.New("webhook secret is required")
	}
	if options.WebhookTolerance < 0 {
		return nil, errors.New("webhook tolerance must not be negative")
	}
	if options.MaxBodyBytes < 1 {
		return nil, errors.New("maximum body size must be positive")
	}
	if options.Store == nil {
		options.Store = store.New()
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &API{
		apiToken:         options.APIToken,
		webhookSecret:    []byte(options.WebhookSecret),
		webhookTolerance: options.WebhookTolerance,
		maxBodyBytes:     options.MaxBodyBytes,
		store:            options.Store,
		now:              options.Now,
	}, nil
}

func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := safeRequestID(r.Header.Get("X-Request-Id"))
	w.Header().Set("X-Request-Id", requestID)
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.URL.Path == "/healthz":
		if r.Method != http.MethodGet {
			a.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestID, nil)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case r.URL.Path == "/v1/orders":
		if r.Method != http.MethodPost {
			a.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestID, nil)
			return
		}
		a.createOrder(w, r, requestID)
	case strings.HasPrefix(r.URL.Path, "/v1/orders/") && !strings.Contains(strings.TrimPrefix(r.URL.Path, "/v1/orders/"), "/"):
		if r.Method != http.MethodGet {
			a.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestID, nil)
			return
		}
		a.getOrder(w, r, strings.TrimPrefix(r.URL.Path, "/v1/orders/"), requestID)
	case r.URL.Path == "/v1/webhooks/order":
		if r.Method != http.MethodPost {
			a.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestID, nil)
			return
		}
		a.receiveWebhook(w, r, requestID)
	default:
		a.writeError(w, http.StatusNotFound, "not_found", "not found", requestID, nil)
	}
}

func (a *API) createOrder(w http.ResponseWriter, r *http.Request, requestID string) {
	if !a.authorized(r.Header.Get("Authorization")) {
		a.writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized", requestID, nil)
		return
	}
	if !jsonContentType(r.Header.Get("Content-Type")) {
		a.writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "unsupported media type", requestID, nil)
		return
	}
	body, ok := a.readBody(w, r, requestID)
	if !ok {
		return
	}
	input, details, malformed := decodeOrder(body)
	if malformed {
		a.writeError(w, http.StatusBadRequest, "bad_request", "malformed JSON", requestID, nil)
		return
	}
	if len(details) != 0 {
		a.writeError(w, http.StatusBadRequest, "validation_error", "request validation failed", requestID, details)
		return
	}
	order := a.store.CreateOrder(input.SKU, input.Quantity, input.CustomerRef, a.now())
	writeJSON(w, http.StatusCreated, order)
}

func (a *API) getOrder(w http.ResponseWriter, r *http.Request, id, requestID string) {
	if !a.authorized(r.Header.Get("Authorization")) {
		a.writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized", requestID, nil)
		return
	}
	if !orderIDPattern.MatchString(id) {
		a.writeError(w, http.StatusBadRequest, "validation_error", "request validation failed", requestID, []fieldDetail{{Field: "id", Message: "must match ord_ followed by 16 lowercase hex characters"}})
		return
	}
	order, ok := a.store.Order(id)
	if !ok {
		a.writeError(w, http.StatusNotFound, "not_found", "not found", requestID, nil)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (a *API) receiveWebhook(w http.ResponseWriter, r *http.Request, requestID string) {
	if !jsonContentType(r.Header.Get("Content-Type")) {
		a.writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "unsupported media type", requestID, nil)
		return
	}
	body, ok := a.readBody(w, r, requestID)
	if !ok {
		return
	}
	if err := webhooksig.Verify(a.webhookSecret, r.Header.Get("Lab-Signature"), body, a.now(), a.webhookTolerance); err != nil {
		a.writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized", requestID, nil)
		return
	}
	event, details, malformed := decodeWebhook(body)
	if malformed {
		a.writeError(w, http.StatusBadRequest, "bad_request", "malformed JSON", requestID, nil)
		return
	}
	if len(details) != 0 {
		a.writeError(w, http.StatusBadRequest, "validation_error", "request validation failed", requestID, details)
		return
	}
	duplicate := a.store.AcceptEvent(event.ID)
	if duplicate {
		writeJSON(w, http.StatusOK, webhookResult{Status: "duplicate", Duplicate: true, EventID: event.ID})
		return
	}
	writeJSON(w, http.StatusAccepted, webhookResult{Status: "accepted", Duplicate: false, EventID: event.ID})
}

func (a *API) authorized(header string) bool {
	expected := []byte("Bearer " + a.apiToken)
	return subtle.ConstantTimeCompare([]byte(header), expected) == 1
}

func (a *API) readBody(w http.ResponseWriter, r *http.Request, requestID string) ([]byte, bool) {
	reader := io.LimitReader(r.Body, a.maxBodyBytes+1)
	body, err := io.ReadAll(reader)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "bad_request", "could not read request body", requestID, nil)
		return nil, false
	}
	if int64(len(body)) > a.maxBodyBytes {
		a.writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "payload too large", requestID, nil)
		return nil, false
	}
	return body, true
}

func (a *API) writeError(w http.ResponseWriter, status int, code, message, requestID string, details []fieldDetail) {
	writeJSON(w, status, errorEnvelope{Error: errorBody{
		Code: code, Message: message, RequestID: requestID, Details: details,
	}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func safeRequestID(candidate string) string {
	if requestIDPattern.MatchString(candidate) {
		return candidate
	}
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "req_0000000000000000"
	}
	return "req_" + hex.EncodeToString(bytes)
}

func jsonContentType(value string) bool {
	mediaType, parameters, err := mime.ParseMediaType(value)
	if err != nil || mediaType != "application/json" {
		return false
	}
	if len(parameters) == 0 {
		return true
	}
	charset, ok := parameters["charset"]
	return ok && len(parameters) == 1 && strings.EqualFold(charset, "utf-8")
}

type orderInput struct {
	SKU         string
	Quantity    int
	CustomerRef string
}

type rawOrder struct {
	SKU         *string         `json:"sku"`
	Quantity    json.RawMessage `json:"quantity"`
	CustomerRef *string         `json:"customer_ref"`
}

func decodeOrder(body []byte) (orderInput, []fieldDetail, bool) {
	var raw rawOrder
	if details, malformed := decodeStrict(body, &raw); details != nil || malformed {
		return orderInput{}, details, malformed
	}
	details := make([]fieldDetail, 0, 3)
	input := orderInput{}
	if raw.SKU == nil {
		details = append(details, fieldDetail{Field: "sku", Message: "is required"})
	} else if !skuPattern.MatchString(*raw.SKU) {
		details = append(details, fieldDetail{Field: "sku", Message: "must use 1 to 64 uppercase letters, digits, underscores, or hyphens"})
	} else {
		input.SKU = *raw.SKU
	}
	if len(raw.Quantity) == 0 || json.Unmarshal(raw.Quantity, &input.Quantity) != nil || input.Quantity < 1 || input.Quantity > 100 {
		details = append(details, fieldDetail{Field: "quantity", Message: "must be an integer from 1 to 100"})
	}
	if raw.CustomerRef == nil {
		details = append(details, fieldDetail{Field: "customer_ref", Message: "is required"})
	} else if !customerRefPattern.MatchString(*raw.CustomerRef) {
		details = append(details, fieldDetail{Field: "customer_ref", Message: "must use 1 to 64 letters, digits, underscores, or hyphens"})
	} else {
		input.CustomerRef = *raw.CustomerRef
	}
	return input, details, false
}

type webhookInput struct {
	ID         string
	Type       string
	OrderID    string
	OccurredAt time.Time
}

type rawWebhook struct {
	ID         *string `json:"id"`
	Type       *string `json:"type"`
	OrderID    *string `json:"order_id"`
	OccurredAt *string `json:"occurred_at"`
}

func decodeWebhook(body []byte) (webhookInput, []fieldDetail, bool) {
	var raw rawWebhook
	if details, malformed := decodeStrict(body, &raw); details != nil || malformed {
		return webhookInput{}, details, malformed
	}
	details := make([]fieldDetail, 0, 4)
	input := webhookInput{}
	if raw.ID == nil {
		details = append(details, fieldDetail{Field: "id", Message: "is required"})
	} else if !eventIDPattern.MatchString(*raw.ID) {
		details = append(details, fieldDetail{Field: "id", Message: "must use 1 to 80 letters, digits, underscores, or hyphens"})
	} else {
		input.ID = *raw.ID
	}
	if raw.Type == nil {
		details = append(details, fieldDetail{Field: "type", Message: "is required"})
	} else if *raw.Type != "order.accepted" {
		details = append(details, fieldDetail{Field: "type", Message: "must be order.accepted"})
	} else {
		input.Type = *raw.Type
	}
	if raw.OrderID == nil {
		details = append(details, fieldDetail{Field: "order_id", Message: "is required"})
	} else if !orderIDPattern.MatchString(*raw.OrderID) {
		details = append(details, fieldDetail{Field: "order_id", Message: "must match ord_ followed by 16 lowercase hex characters"})
	} else {
		input.OrderID = *raw.OrderID
	}
	if raw.OccurredAt == nil {
		details = append(details, fieldDetail{Field: "occurred_at", Message: "is required"})
	} else if occurredAt, err := time.Parse(time.RFC3339, *raw.OccurredAt); err != nil {
		details = append(details, fieldDetail{Field: "occurred_at", Message: "must be an RFC3339 timestamp"})
	} else {
		input.OccurredAt = occurredAt
	}
	return input, details, false
}

func decodeStrict(body []byte, target any) ([]fieldDetail, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var typeError *json.UnmarshalTypeError
		if errors.As(err, &typeError) {
			field := typeError.Field
			if field == "" {
				field = "request"
			}
			return []fieldDetail{{Field: field, Message: "has an invalid type"}}, false
		}
		const unknownPrefix = "json: unknown field "
		if strings.HasPrefix(err.Error(), unknownPrefix) {
			field := strings.Trim(err.Error()[len(unknownPrefix):], `"`)
			return []fieldDetail{{Field: field, Message: "is not allowed"}}, false
		}
		return nil, true
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, true
	}
	return nil, false
}
