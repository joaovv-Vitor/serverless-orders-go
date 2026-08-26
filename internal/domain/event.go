package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// OrderCreatedEventType is the stable name used to route and identify this event.
	OrderCreatedEventType = "OrderCreated"
	// OrderCreatedEventVersion identifies the current JSON contract version.
	OrderCreatedEventVersion = 1
)

// OrderCreatedEvent is the version 1 integration event emitted for an accepted order.
// It is intentionally separate from HTTP and internal order models so that it can
// evolve as a public contract without coupling those models to consumers.
type OrderCreatedEvent struct {
	EventID      string           `json:"eventId"`
	EventType    string           `json:"eventType"`
	EventVersion int              `json:"eventVersion"`
	OccurredAt   time.Time        `json:"occurredAt"`
	Data         OrderCreatedData `json:"data"`
}

// OrderCreatedData is the payload carried by OrderCreatedEvent version 1.
type OrderCreatedData struct {
	OrderID    string             `json:"orderId"`
	CustomerID string             `json:"customerId"`
	Items      []OrderCreatedItem `json:"items"`
}

// OrderCreatedItem is the event representation of an order item.
type OrderCreatedItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// NewOrderCreatedEvent builds a version 1 event from validated order data.
func NewOrderCreatedEvent(
	eventID string,
	occurredAt time.Time,
	orderID string,
	order CreateOrderInput,
) (OrderCreatedEvent, error) {
	items := make([]OrderCreatedItem, len(order.Items))
	for index, item := range order.Items {
		items[index] = OrderCreatedItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	event := OrderCreatedEvent{
		EventID:      eventID,
		EventType:    OrderCreatedEventType,
		EventVersion: OrderCreatedEventVersion,
		OccurredAt:   occurredAt.UTC(),
		Data: OrderCreatedData{
			OrderID:    orderID,
			CustomerID: order.CustomerID,
			Items:      items,
		},
	}

	if err := event.Validate(); err != nil {
		return OrderCreatedEvent{}, err
	}
	return event, nil
}

// Validate checks the required fields and fixed metadata of version 1.
func (event OrderCreatedEvent) Validate() error {
	if strings.TrimSpace(event.EventID) == "" {
		return errors.New("eventId is required")
	}
	if event.EventType != OrderCreatedEventType {
		return fmt.Errorf("eventType must be %q", OrderCreatedEventType)
	}
	if event.EventVersion != OrderCreatedEventVersion {
		return fmt.Errorf("eventVersion must be %d", OrderCreatedEventVersion)
	}
	if event.OccurredAt.IsZero() {
		return errors.New("occurredAt is required")
	}
	if strings.TrimSpace(event.Data.OrderID) == "" {
		return errors.New("data.orderId is required")
	}

	order := CreateOrderInput{
		CustomerID: event.Data.CustomerID,
		Items:      make([]OrderItem, len(event.Data.Items)),
	}
	for index, item := range event.Data.Items {
		order.Items[index] = OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	if err := order.Validate(); err != nil {
		return fmt.Errorf("data: %w", err)
	}

	return nil
}
