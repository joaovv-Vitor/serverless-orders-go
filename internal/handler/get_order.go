package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/orders"
)

const getOrderService = "get-order"

// OrderReader is the persistence capability required by GET /orders/{id}.
type OrderReader interface {
	Get(context.Context, string) (domain.Order, error)
}

// GetOrderHandler returns the latest persisted representation of an order.
type GetOrderHandler struct {
	orders OrderReader
	logger *slog.Logger
}

// GetOrderResponse is the HTTP representation of a persisted order.
type GetOrderResponse struct {
	OrderID    string                 `json:"orderId"`
	CustomerID string                 `json:"customerId"`
	Items      []GetOrderItemResponse `json:"items"`
	Status     domain.OrderStatus     `json:"status"`
	CreatedAt  string                 `json:"createdAt"`
	UpdatedAt  string                 `json:"updatedAt"`
}

// GetOrderItemResponse is an item returned by GET /orders/{id}.
type GetOrderItemResponse struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// NewGetOrderHandler creates the order query handler.
func NewGetOrderHandler(repository OrderReader, logger *slog.Logger) GetOrderHandler {
	return GetOrderHandler{orders: repository, logger: logger}
}

// Handle retrieves one order using the API Gateway path parameter.
func (handler GetOrderHandler) Handle(
	ctx context.Context,
	request events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {
	orderID := strings.TrimSpace(request.PathParameters["id"])
	if orderID == "" {
		return jsonResponse(http.StatusBadRequest, errorResponse{Message: "order id is required"})
	}

	order, err := handler.orders.Get(ctx, orderID)
	if errors.Is(err, orders.ErrNotFound) {
		handler.logger.InfoContext(ctx, "order not found",
			"service", getOrderService,
			"orderId", orderID,
		)
		return jsonResponse(http.StatusNotFound, errorResponse{Message: "order not found"})
	}
	if err != nil {
		handler.logger.ErrorContext(ctx, "order retrieval failed",
			"service", getOrderService,
			"orderId", orderID,
			"error", err,
		)
		return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("get order %q: %w", orderID, err)
	}

	handler.logger.InfoContext(ctx, "order retrieved",
		"service", getOrderService,
		"orderId", order.OrderID,
		"status", order.Status,
	)
	return jsonResponse(http.StatusOK, newGetOrderResponse(order))
}

func newGetOrderResponse(order domain.Order) GetOrderResponse {
	items := make([]GetOrderItemResponse, len(order.Items))
	for index, item := range order.Items {
		items[index] = GetOrderItemResponse{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	return GetOrderResponse{
		OrderID:    order.OrderID,
		CustomerID: order.CustomerID,
		Items:      items,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:  order.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
