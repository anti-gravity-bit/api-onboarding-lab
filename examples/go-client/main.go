package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anti-gravity-bit/api-onboarding-lab/webhooksig"
)

type order struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Quantity    int       `json:"quantity"`
	CustomerRef string    `json:"customer_ref"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type webhookResult struct {
	Status    string `json:"status"`
	Duplicate bool   `json:"duplicate"`
	EventID   string `json:"event_id"`
}

func main() {
	if err := run(); err != nil {
		log.New(os.Stderr, "go-client: ", 0).Fatal(err)
	}
}

func run() error {
	apiToken := os.Getenv("API_TOKEN")
	webhookSecret := os.Getenv("WEBHOOK_SECRET")
	if apiToken == "" || webhookSecret == "" {
		return errors.New("API_TOKEN and WEBHOOK_SECRET are required")
	}
	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	client := &http.Client{Timeout: 10 * time.Second}

	created, err := createOrder(client, baseURL, apiToken)
	if err != nil {
		return err
	}
	if _, err := fetchOrder(client, baseURL, apiToken, created.ID); err != nil {
		return err
	}
	result, err := sendWebhook(client, baseURL, webhookSecret, created.ID)
	if err != nil {
		return err
	}

	output := struct {
		Order   order         `json:"order"`
		Webhook webhookResult `json:"webhook"`
	}{Order: created, Webhook: result}
	return json.NewEncoder(os.Stdout).Encode(output)
}

func createOrder(client *http.Client, baseURL, token string) (order, error) {
	body := struct {
		SKU         string `json:"sku"`
		Quantity    int    `json:"quantity"`
		CustomerRef string `json:"customer_ref"`
	}{SKU: "DEMO-WIDGET", Quantity: 2, CustomerRef: "cust_demo_1001"}
	var created order
	if err := doJSON(client, http.MethodPost, baseURL+"/v1/orders", token, "", body, http.StatusCreated, &created); err != nil {
		return order{}, fmt.Errorf("create order: %w", err)
	}
	return created, nil
}

func fetchOrder(client *http.Client, baseURL, token, id string) (order, error) {
	var fetched order
	if err := doJSON(client, http.MethodGet, baseURL+"/v1/orders/"+id, token, "", nil, http.StatusOK, &fetched); err != nil {
		return order{}, fmt.Errorf("fetch order: %w", err)
	}
	return fetched, nil
}

func sendWebhook(client *http.Client, baseURL, secret, orderID string) (webhookResult, error) {
	now := time.Now().UTC()
	event := struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		OrderID    string `json:"order_id"`
		OccurredAt string `json:"occurred_at"`
	}{
		ID:         fmt.Sprintf("evt_go_client_%d", now.UnixNano()),
		Type:       "order.accepted",
		OrderID:    orderID,
		OccurredAt: now.Format(time.RFC3339),
	}
	body, err := json.Marshal(event)
	if err != nil {
		return webhookResult{}, fmt.Errorf("encode webhook: %w", err)
	}
	signature := webhooksig.Sign([]byte(secret), now.Unix(), body)
	var result webhookResult
	if err := doRawJSON(client, http.MethodPost, baseURL+"/v1/webhooks/order", "", signature, body, http.StatusAccepted, &result); err != nil {
		return webhookResult{}, fmt.Errorf("send webhook: %w", err)
	}
	return result, nil
}

func doJSON(client *http.Client, method, url, token, signature string, input any, expectedStatus int, output any) error {
	var body []byte
	var err error
	if input != nil {
		body, err = json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}
	return doRawJSON(client, method, url, token, signature, body, expectedStatus, output)
}

func doRawJSON(client *http.Client, method, url, token, signature string, body []byte, expectedStatus int, output any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if len(body) != 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if signature != "" {
		req.Header.Set("Lab-Signature", signature)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	closeErr := resp.Body.Close()
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close response: %w", closeErr)
	}
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if output != nil {
		if err := json.Unmarshal(responseBody, output); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
