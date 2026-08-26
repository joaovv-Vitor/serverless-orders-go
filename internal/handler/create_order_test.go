package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestCreateOrderHandlerHandle(t *testing.T) {
	t.Parallel()

	handler := CreateOrderHandler{
		newOrderID: func() (string, error) { return "order-test-id", nil },
	}

	tests := []struct {
		name        string
		request     events.APIGatewayV2HTTPRequest
		wantStatus  int
		wantPayload map[string]string
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
		})
	}
}

func TestCreateOrderHandlerHandleReturnsIDGenerationError(t *testing.T) {
	t.Parallel()

	handler := CreateOrderHandler{
		newOrderID: func() (string, error) { return "", errors.New("random source unavailable") },
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

func TestGenerateOrderID(t *testing.T) {
	t.Parallel()

	first, err := generateOrderID()
	if err != nil {
		t.Fatalf("generateOrderID() error = %v", err)
	}
	second, err := generateOrderID()
	if err != nil {
		t.Fatalf("generateOrderID() second error = %v", err)
	}

	if len(first) != 36 {
		t.Fatalf("generateOrderID() length = %d, want 36", len(first))
	}
	if first == second {
		t.Fatal("generateOrderID() returned the same ID twice")
	}
}
