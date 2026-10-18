package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var fallbackID atomic.Uint64

// Order is a synthetic order retained in process memory.
type Order struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Quantity    int       `json:"quantity"`
	CustomerRef string    `json:"customer_ref"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Memory stores orders and accepted webhook IDs until the process exits.
type Memory struct {
	mu       sync.RWMutex
	orders   map[string]Order
	eventIDs map[string]struct{}
}

func New() *Memory {
	return &Memory{
		orders:   make(map[string]Order),
		eventIDs: make(map[string]struct{}),
	}
}

func (m *Memory) CreateOrder(sku string, quantity int, customerRef string, createdAt time.Time) Order {
	id := randomID("ord_")
	order := Order{
		ID:          id,
		SKU:         sku,
		Quantity:    quantity,
		CustomerRef: customerRef,
		Status:      "accepted",
		CreatedAt:   createdAt.UTC(),
	}
	m.mu.Lock()
	m.orders[id] = order
	m.mu.Unlock()
	return order
}

func (m *Memory) Order(id string) (Order, bool) {
	m.mu.RLock()
	order, ok := m.orders[id]
	m.mu.RUnlock()
	return order, ok
}

// AcceptEvent atomically records an event ID and reports whether it was
// already present.
func (m *Memory) AcceptEvent(id string) (duplicate bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.eventIDs[id]; ok {
		return true
	}
	m.eventIDs[id] = struct{}{}
	return false
}

func (m *Memory) EventCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.eventIDs)
}

func randomID(prefix string) string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s%016x", prefix, fallbackID.Add(1))
	}
	return prefix + hex.EncodeToString(bytes)
}
