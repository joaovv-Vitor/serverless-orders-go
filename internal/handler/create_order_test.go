package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

type eventPublisherFunc func(context.Context, domain.OrderCreatedEvent) error

func (function eventPublisherFunc) Publish(ctx context.Context, event domain.OrderCreatedEvent) error {
	return function(ctx, event)
}

func TestCreateOrderHandlerHandle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		request       events.APIGatewayV2HTTPRequest
		wantStatus    int
		wantPayload   map[string]string
		wantPublished bool
	}{
		{
			name: "accepts a valid order",
			request: events.APIGatewayV2HTTPRequest{Body: `{
				"customerId":"customer-123",
				"items":[{"productId":"product-456","quantity":2}]
			}`},
			wantStatus: http.StatusAccepted,
			wantPayload: map[string]string{
				"orderId": "order-test-id",
				"status":  "accepted",
			},
			wantPublished: true,
		},
		{
			name: "accepts a base64 encoded body",
			request: events.APIGatewayV2HTTPRequest{
				Body: base64.StdEncoding.EncodeToString([]byte(`{
					"customerId":"customer-123",
					"items":[{"productId":"product-456","quantity":2}]
				}`)),
				IsBase64Encoded: true,
			},
			wantStatus: http.StatusAccepted,
			wantPayload: map[string]string{
				"orderId": "order-test-id",
				"status":  "accepted",
			},
			wantPublished: true,
		},
		{
			name:        "rejects an empty body",
			request:     events.APIGatewayV2HTTPRequest{},
			wantStatus:  http.StatusBadRequest,
			wantPayload: map[string]string{"message": "request body is required"},
		},
		{
			name:        "rejects malformed JSON",
			request:     events.APIGatewayV2HTTPRequest{Body: `{"customerId":`},
			wantStatus:  http.StatusBadRequest,
			wantPayload: map[string]string{"message": "request body must be valid JSON"},
		},
		{
			name: "rejects an unknown field",
			request: events.APIGatewayV2HTTPRequest{Body: `{
				"customerId":"customer-123",
				"items":[{"productId":"product-456","quantity":2}],
				"unexpected":true
			}`},
			wantStatus:  http.StatusBadRequest,
			wantPayload: map[string]string{"message": "request body must be valid JSON"},
		},
		{
			name: "rejects an invalid order",
			request: events.APIGatewayV2HTTPRequest{Body: `{
				"customerId":"customer-123",
				"items":[]
			}`},
			wantStatus:  http.StatusBadRequest,
			wantPayload: map[string]string{"message": "at least one item is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var publishedEvent domain.OrderCreatedEvent
			handler := CreateOrderHandler{
				publisher: eventPublisherFunc(func(_ context.Context, event domain.OrderCreatedEvent) error {
					publishedEvent = event
					return nil
				}),
				newOrderID: func() (string, error) { return "order-test-id", nil },
				newEventID: func() (string, error) { return "event-test-id", nil },
				now: func() time.Time {
					return time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC)
				},
			}

			response, err := handler.Handle(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if response.StatusCode != tt.wantStatus {
				t.Fatalf("Handle() status = %d, want %d", response.StatusCode, tt.wantStatus)
			}
			if response.Headers["content-type"] != contentTypeJSON {
				t.Fatalf("Handle() content-type = %q", response.Headers["content-type"])
			}

			var payload map[string]string
			if err := json.Unmarshal([]byte(response.Body), &payload); err != nil {
				t.Fatalf("response body is not valid JSON: %v", err)
			}
			if len(payload) != len(tt.wantPayload) {
				t.Fatalf("response payload = %#v, want %#v", payload, tt.wantPayload)
			}
			for key, want := range tt.wantPayload {
				if payload[key] != want {
					t.Fatalf("response payload[%q] = %q, want %q", key, payload[key], want)
				}
			}

			published := publishedEvent.EventID != ""
			if published != tt.wantPublished {
				t.Fatalf("event published = %v, want %v", published, tt.wantPublished)
			}
			if tt.wantPublished {
				if publishedEvent.EventID != "event-test-id" {
					t.Fatalf("published eventId = %q", publishedEvent.EventID)
				}
				if publishedEvent.Data.OrderID != "order-test-id" {
					t.Fatalf("published orderId = %q", publishedEvent.Data.OrderID)
				}
				if publishedEvent.EventType != domain.OrderCreatedEventType {
					t.Fatalf("published eventType = %q", publishedEvent.EventType)
				}
			}
		})
	}
}

func TestCreateOrderHandlerHandleReturnsIDGenerationError(t *testing.T) {
	t.Parallel()

	handler := CreateOrderHandler{
		publisher:  eventPublisherFunc(func(context.Context, domain.OrderCreatedEvent) error { return nil }),
		newOrderID: func() (string, error) { return "", errors.New("random source unavailable") },
		newEventID: func() (string, error) { return "event-test-id", nil },
		now:        time.Now,
	}
	request := events.APIGatewayV2HTTPRequest{Body: `{
		"customerId":"customer-123",
		"items":[{"productId":"product-456","quantity":2}]
	}`}

	_, err := handler.Handle(context.Background(), request)
	if err == nil || err.Error() != "generate order id: random source unavailable" {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestCreateOrderHandlerHandleReturnsEventIDGenerationError(t *testing.T) {
	t.Parallel()

	handler := CreateOrderHandler{
		publisher:  eventPublisherFunc(func(context.Context, domain.OrderCreatedEvent) error { return nil }),
		newOrderID: func() (string, error) { return "order-test-id", nil },
		newEventID: func() (string, error) { return "", errors.New("random source unavailable") },
		now:        time.Now,
	}
	request := validCreateOrderAPIRequest()

	_, err := handler.Handle(context.Background(), request)
	if err == nil || err.Error() != "generate event id: random source unavailable" {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestCreateOrderHandlerHandleReturnsPublishError(t *testing.T) {
	t.Parallel()

	handler := CreateOrderHandler{
		publisher: eventPublisherFunc(func(context.Context, domain.OrderCreatedEvent) error {
			return errors.New("SNS unavailable")
		}),
		newOrderID: func() (string, error) { return "order-test-id", nil },
		newEventID: func() (string, error) { return "event-test-id", nil },
		now:        time.Now,
	}

	_, err := handler.Handle(context.Background(), validCreateOrderAPIRequest())
	if err == nil || err.Error() != "publish OrderCreated event: SNS unavailable" {
		t.Fatalf("Handle() error = %v", err)
	}
}

func validCreateOrderAPIRequest() events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{Body: `{
		"customerId":"customer-123",
		"items":[{"productId":"product-456","quantity":2}]
	}`}
}

func TestGenerateID(t *testing.T) {
	t.Parallel()

	first, err := generateID()
	if err != nil {
		t.Fatalf("generateID() error = %v", err)
	}
	second, err := generateID()
	if err != nil {
		t.Fatalf("generateID() second error = %v", err)
	}

	if len(first) != 36 {
		t.Fatalf("generateID() length = %d, want 36", len(first))
	}
	if first == second {
		t.Fatal("generateID() returned the same ID twice")
	}
}
