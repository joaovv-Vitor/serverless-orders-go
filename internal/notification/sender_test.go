package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

func TestSenderSendWritesStructuredLog(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	sender, err := NewSender(slog.New(slog.NewJSONHandler(&output, nil)), "")
	if err != nil {
		t.Fatalf("NewSender() error = %v", err)
	}
	event := notificationEvent(t)

	if err := sender.Send(context.Background(), event); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("log is not valid JSON: %v", err)
	}
	if entry["msg"] != "notification sent" || entry["service"] != serviceName {
		t.Fatalf("notification log = %#v", entry)
	}
	if entry["eventId"] != event.EventID || entry["orderId"] != event.Data.OrderID {
		t.Fatalf("notification log missing correlation fields: %#v", entry)
	}
	if entry["customerId"] != event.Data.CustomerID {
		t.Fatalf("notification customerId = %#v", entry["customerId"])
	}
}

func TestNewSenderRequiresLogger(t *testing.T) {
	t.Parallel()

	_, err := NewSender(nil, "")
	if err == nil || err.Error() != "logger is required" {
		t.Fatalf("NewSender() error = %v", err)
	}
}

func TestSenderSendCanForceFailure(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	sender, err := NewSender(slog.New(slog.NewJSONHandler(&output, nil)), "customer-123")
	if err != nil {
		t.Fatalf("NewSender() error = %v", err)
	}

	err = sender.Send(context.Background(), notificationEvent(t))
	if err == nil || err.Error() != "forced notification failure" {
		t.Fatalf("Send() error = %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("failure log is not valid JSON: %v", err)
	}
	if entry["level"] != "ERROR" || entry["msg"] != "forced notification failure" {
		t.Fatalf("failure log = %#v", entry)
	}
}

func notificationEvent(t *testing.T) domain.OrderCreatedEvent {
	t.Helper()

	event, err := domain.NewOrderCreatedEvent(
		"event-123",
		time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC),
		"order-123",
		domain.CreateOrderInput{
			CustomerID: "customer-123",
			Items:      []domain.OrderItem{{ProductID: "product-456", Quantity: 2}},
		},
	)
	if err != nil {
		t.Fatalf("NewOrderCreatedEvent() error = %v", err)
	}
	return event
}
