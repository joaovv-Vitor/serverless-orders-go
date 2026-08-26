package stock

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

func TestProcessorProcessWritesStructuredLogs(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	processor, err := NewProcessor(slog.New(slog.NewJSONHandler(&output, nil)))
	if err != nil {
		t.Fatalf("NewProcessor() error = %v", err)
	}
	event := stockEvent(t)

	if err := processor.Process(context.Background(), event); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	var entries []map[string]any
	scanner := bufio.NewScanner(&output)
	for scanner.Scan() {
		var entry map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("log line is not valid JSON: %v", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan logs: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("log entries = %d, want 3", len(entries))
	}
	for index, entry := range entries {
		if entry["service"] != serviceName || entry["eventId"] != event.EventID || entry["orderId"] != event.Data.OrderID {
			t.Fatalf("log entry %d missing correlation fields: %#v", index, entry)
		}
	}
	if entries[0]["productId"] != "product-456" || entries[0]["quantity"] != float64(2) {
		t.Fatalf("first item log = %#v", entries[0])
	}
	if entries[1]["productId"] != "product-789" || entries[1]["quantity"] != float64(1) {
		t.Fatalf("second item log = %#v", entries[1])
	}
	if entries[2]["msg"] != "stock processed" {
		t.Fatalf("completion log = %#v", entries[2])
	}
}

func TestNewProcessorRequiresLogger(t *testing.T) {
	t.Parallel()

	_, err := NewProcessor(nil)
	if err == nil || err.Error() != "logger is required" {
		t.Fatalf("NewProcessor() error = %v", err)
	}
}

func stockEvent(t *testing.T) domain.OrderCreatedEvent {
	t.Helper()

	event, err := domain.NewOrderCreatedEvent(
		"event-123",
		time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC),
		"order-123",
		domain.CreateOrderInput{
			CustomerID: "customer-123",
			Items: []domain.OrderItem{
				{ProductID: "product-456", Quantity: 2},
				{ProductID: "product-789", Quantity: 1},
			},
		},
	)
	if err != nil {
		t.Fatalf("NewOrderCreatedEvent() error = %v", err)
	}
	return event
}
