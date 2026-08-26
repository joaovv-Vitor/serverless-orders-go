package domain

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestNewOrderCreatedEvent(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.August, 26, 12, 30, 0, 0, time.FixedZone("BRT", -3*60*60))
	order := CreateOrderInput{
		CustomerID: "customer-123",
		Items: []OrderItem{
			{ProductID: "product-456", Quantity: 2},
		},
	}

	event, err := NewOrderCreatedEvent("event-123", occurredAt, "order-123", order)
	if err != nil {
		t.Fatalf("NewOrderCreatedEvent() error = %v", err)
	}

	if event.EventType != OrderCreatedEventType {
		t.Fatalf("EventType = %q, want %q", event.EventType, OrderCreatedEventType)
	}
	if event.EventVersion != OrderCreatedEventVersion {
		t.Fatalf("EventVersion = %d, want %d", event.EventVersion, OrderCreatedEventVersion)
	}
	if event.OccurredAt.Location() != time.UTC {
		t.Fatalf("OccurredAt location = %v, want UTC", event.OccurredAt.Location())
	}
	if event.Data.OrderID != "order-123" || event.Data.CustomerID != order.CustomerID {
		t.Fatalf("Data = %#v", event.Data)
	}
	if len(event.Data.Items) != 1 || event.Data.Items[0].ProductID != "product-456" {
		t.Fatalf("Data.Items = %#v", event.Data.Items)
	}

	// The contract owns its item slice; later changes to internal input cannot
	// silently alter an event that has already been created.
	order.Items[0].ProductID = "changed"
	if event.Data.Items[0].ProductID != "product-456" {
		t.Fatal("event items changed after mutating the source order")
	}
}

func TestOrderCreatedEventJSONContract(t *testing.T) {
	t.Parallel()

	event, err := NewOrderCreatedEvent(
		"event-123",
		time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC),
		"order-123",
		CreateOrderInput{
			CustomerID: "customer-123",
			Items: []OrderItem{
				{ProductID: "product-456", Quantity: 2},
			},
		},
	)
	if err != nil {
		t.Fatalf("NewOrderCreatedEvent() error = %v", err)
	}

	got, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	want := `{"eventId":"event-123","eventType":"OrderCreated","eventVersion":1,"occurredAt":"2026-08-26T15:30:00Z","data":{"orderId":"order-123","customerId":"customer-123","items":[{"productId":"product-456","quantity":2}]}}`
	if string(got) != want {
		t.Fatalf("JSON contract = %s, want %s", got, want)
	}
}

func TestOrderCreatedEventExample(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("../../events/order-created.json")
	if err != nil {
		t.Fatalf("read event example: %v", err)
	}

	var event OrderCreatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		t.Fatalf("event example is not valid JSON: %v", err)
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("event example does not satisfy the v1 contract: %v", err)
	}
}

func TestOrderCreatedEventValidate(t *testing.T) {
	t.Parallel()

	validEvent, err := NewOrderCreatedEvent(
		"event-123",
		time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC),
		"order-123",
		CreateOrderInput{
			CustomerID: "customer-123",
			Items:      []OrderItem{{ProductID: "product-456", Quantity: 2}},
		},
	)
	if err != nil {
		t.Fatalf("NewOrderCreatedEvent() error = %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*OrderCreatedEvent)
		wantErr string
	}{
		{name: "valid event", mutate: func(*OrderCreatedEvent) {}},
		{
			name:    "missing event id",
			mutate:  func(event *OrderCreatedEvent) { event.EventID = "" },
			wantErr: "eventId is required",
		},
		{
			name:    "wrong event type",
			mutate:  func(event *OrderCreatedEvent) { event.EventType = "OrderUpdated" },
			wantErr: `eventType must be "OrderCreated"`,
		},
		{
			name:    "unsupported version",
			mutate:  func(event *OrderCreatedEvent) { event.EventVersion = 2 },
			wantErr: "eventVersion must be 1",
		},
		{
			name:    "missing timestamp",
			mutate:  func(event *OrderCreatedEvent) { event.OccurredAt = time.Time{} },
			wantErr: "occurredAt is required",
		},
		{
			name:    "missing order id",
			mutate:  func(event *OrderCreatedEvent) { event.Data.OrderID = "" },
			wantErr: "data.orderId is required",
		},
		{
			name:    "invalid event item",
			mutate:  func(event *OrderCreatedEvent) { event.Data.Items[0].Quantity = 0 },
			wantErr: "data: items[0].quantity must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event := validEvent
			event.Data.Items = append([]OrderCreatedItem(nil), validEvent.Data.Items...)
			tt.mutate(&event)
			err := event.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
