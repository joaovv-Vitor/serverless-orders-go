package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/orders"
)

type fakeOrderReader struct {
	order domain.Order
	err   error
	id    string
}

func (reader *fakeOrderReader) Get(_ context.Context, orderID string) (domain.Order, error) {
	reader.id = orderID
	return reader.order, reader.err
}

func TestGetOrderHandlerReturnsOrder(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC)
	reader := &fakeOrderReader{order: domain.Order{
		OrderID:    "order-123",
		CustomerID: "customer-123",
		Items:      []domain.OrderItem{{ProductID: "product-456", Quantity: 2}},
		Status:     domain.OrderStatusProcessed,
		CreatedAt:  now,
		UpdatedAt:  now.Add(time.Minute),
	}}
	handler := NewGetOrderHandler(reader, discardLogger())

	response, err := handler.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		PathParameters: map[string]string{"id": "order-123"},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var payload GetOrderResponse
	if err := json.Unmarshal([]byte(response.Body), &payload); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if payload.OrderID != "order-123" || payload.Status != domain.OrderStatusProcessed {
		t.Fatalf("payload = %#v", payload)
	}
	if len(payload.Items) != 1 || payload.Items[0].ProductID != "product-456" {
		t.Fatalf("items = %#v", payload.Items)
	}
}

func TestGetOrderHandlerReturnsNotFound(t *testing.T) {
	t.Parallel()

	handler := NewGetOrderHandler(&fakeOrderReader{err: orders.ErrNotFound}, discardLogger())
	response, err := handler.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		PathParameters: map[string]string{"id": "missing"},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusNotFound || response.Body != `{"message":"order not found"}` {
		t.Fatalf("response = %#v", response)
	}
}

func TestGetOrderHandlerRequiresID(t *testing.T) {
	t.Parallel()

	reader := &fakeOrderReader{}
	handler := NewGetOrderHandler(reader, discardLogger())
	response, err := handler.Handle(context.Background(), events.APIGatewayV2HTTPRequest{})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusBadRequest || reader.id != "" {
		t.Fatalf("response = %#v, reader id = %q", response, reader.id)
	}
}

func TestGetOrderHandlerReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	handler := NewGetOrderHandler(&fakeOrderReader{err: errors.New("DynamoDB unavailable")}, discardLogger())
	_, err := handler.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		PathParameters: map[string]string{"id": "order-123"},
	})
	if err == nil || err.Error() != `get order "order-123": DynamoDB unavailable` {
		t.Fatalf("Handle() error = %v", err)
	}
}
