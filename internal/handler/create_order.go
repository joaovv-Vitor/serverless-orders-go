package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

const contentTypeJSON = "application/json"

// CreateOrderResponse is the HTTP response returned when an order is accepted.
type CreateOrderResponse struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
}

// CreateOrderRequest is the HTTP contract accepted by POST /orders.
type CreateOrderRequest struct {
	CustomerID string                   `json:"customerId"`
	Items      []CreateOrderItemRequest `json:"items"`
}

// CreateOrderItemRequest is the HTTP representation of an order item.
type CreateOrderItemRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type errorResponse struct {
	Message string `json:"message"`
}

// EventPublisher is the event publishing capability required by CreateOrder.
type EventPublisher interface {
	Publish(context.Context, domain.OrderCreatedEvent) error
}

// CreateOrderHandler handles API Gateway requests for POST /orders.
type CreateOrderHandler struct {
	publisher  EventPublisher
	newOrderID func() (string, error)
	newEventID func() (string, error)
	now        func() time.Time
}

// NewCreateOrderHandler creates a handler with production ID and clock dependencies.
func NewCreateOrderHandler(publisher EventPublisher) CreateOrderHandler {
	return CreateOrderHandler{
		publisher:  publisher,
		newOrderID: generateID,
		newEventID: generateID,
		now:        time.Now,
	}
}

// Handle validates an order, publishes OrderCreated, and returns 202 Accepted.
func (handler CreateOrderHandler) Handle(
	ctx context.Context,
	request events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {
	body, err := requestBody(request)
	if err != nil {
		return jsonResponse(http.StatusBadRequest, errorResponse{Message: err.Error()})
	}

	var requestPayload CreateOrderRequest
	if err := decodeJSON(body, &requestPayload); err != nil {
		return jsonResponse(http.StatusBadRequest, errorResponse{Message: "request body must be valid JSON"})
	}
	input := requestPayload.toDomain()
	if err := input.Validate(); err != nil {
		return jsonResponse(http.StatusBadRequest, errorResponse{Message: err.Error()})
	}

	orderID, err := handler.newOrderID()
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("generate order id: %w", err)
	}
	eventID, err := handler.newEventID()
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("generate event id: %w", err)
	}
	event, err := domain.NewOrderCreatedEvent(eventID, handler.now(), orderID, input)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("create OrderCreated event: %w", err)
	}
	if err := handler.publisher.Publish(ctx, event); err != nil {
		return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("publish OrderCreated event: %w", err)
	}

	return jsonResponse(http.StatusAccepted, CreateOrderResponse{
		OrderID: orderID,
		Status:  "accepted",
	})
}

func (request CreateOrderRequest) toDomain() domain.CreateOrderInput {
	items := make([]domain.OrderItem, len(request.Items))
	for index, item := range request.Items {
		items[index] = domain.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	return domain.CreateOrderInput{
		CustomerID: request.CustomerID,
		Items:      items,
	}
}

func requestBody(request events.APIGatewayV2HTTPRequest) ([]byte, error) {
	if request.Body == "" {
		return nil, errors.New("request body is required")
	}
	if !request.IsBase64Encoded {
		return []byte(request.Body), nil
	}

	body, err := base64.StdEncoding.DecodeString(request.Body)
	if err != nil {
		return nil, errors.New("request body must be valid base64")
	}
	return body, nil
}

func decodeJSON(body []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON value")
	}
	return nil
}

func jsonResponse(statusCode int, payload any) (events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("encode response: %w", err)
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"content-type": contentTypeJSON},
		Body:       string(body),
	}, nil
}

func generateID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}

	// UUID version 4 and RFC 4122 variant bits.
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16]), nil
}
