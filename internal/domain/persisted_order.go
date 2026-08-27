package domain

import (
	"errors"
	"strings"
	"time"
)

// OrderStatus describes the educational lifecycle exposed by GET /orders/{id}.
type OrderStatus string

const (
	OrderStatusAccepted   OrderStatus = "ACCEPTED"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
	OrderStatusFailed     OrderStatus = "FAILED"
)

// Valid reports whether status belongs to the supported order lifecycle.
func (status OrderStatus) Valid() bool {
	switch status {
	case OrderStatusAccepted, OrderStatusProcessing, OrderStatusProcessed, OrderStatusFailed:
		return true
	default:
		return false
	}
}

// Order is the persisted representation of an accepted order.
type Order struct {
	OrderID    string
	CustomerID string
	Items      []OrderItem
	Status     OrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewOrder creates an accepted order ready for persistence.
func NewOrder(orderID string, input CreateOrderInput, now time.Time) (Order, error) {
	if strings.TrimSpace(orderID) == "" {
		return Order{}, errors.New("orderId is required")
	}
	if err := input.Validate(); err != nil {
		return Order{}, err
	}
	if now.IsZero() {
		return Order{}, errors.New("order timestamp is required")
	}

	return Order{
		OrderID:    orderID,
		CustomerID: input.CustomerID,
		Items:      append([]OrderItem(nil), input.Items...),
		Status:     OrderStatusAccepted,
		CreatedAt:  now.UTC(),
		UpdatedAt:  now.UTC(),
	}, nil
}
