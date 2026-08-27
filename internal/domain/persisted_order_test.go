package domain

import (
	"testing"
	"time"
)

func TestNewOrderCreatesAcceptedOrder(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 26, 15, 30, 0, 0, time.FixedZone("test", -3*60*60))
	order, err := NewOrder("order-123", CreateOrderInput{
		CustomerID: "customer-123",
		Items:      []OrderItem{{ProductID: "product-456", Quantity: 2}},
	}, now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if order.Status != OrderStatusAccepted {
		t.Fatalf("status = %q", order.Status)
	}
	if order.CreatedAt.Location() != time.UTC || !order.CreatedAt.Equal(now) || !order.UpdatedAt.Equal(now) {
		t.Fatalf("timestamps = %v / %v", order.CreatedAt, order.UpdatedAt)
	}
}

func TestOrderStatusValid(t *testing.T) {
	t.Parallel()

	for _, status := range []OrderStatus{OrderStatusAccepted, OrderStatusProcessing, OrderStatusProcessed, OrderStatusFailed} {
		if !status.Valid() {
			t.Fatalf("status %q is invalid", status)
		}
	}
	if OrderStatus("UNKNOWN").Valid() {
		t.Fatal("UNKNOWN status is valid")
	}
}
