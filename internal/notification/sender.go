package notification

import (
	"context"
	"errors"
	"log/slog"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

const serviceName = "send-notification"

// Sender simulates customer notification through a structured log.
type Sender struct {
	logger            *slog.Logger
	failureCustomerID string
}

// NewSender creates the notification sender.
func NewSender(logger *slog.Logger, failureCustomerID string) (Sender, error) {
	if logger == nil {
		return Sender{}, errors.New("logger is required")
	}
	return Sender{logger: logger, failureCustomerID: failureCustomerID}, nil
}

// Send records the notification that would be delivered to the customer.
func (sender Sender) Send(ctx context.Context, event domain.OrderCreatedEvent) error {
	if sender.failureCustomerID != "" && event.Data.CustomerID == sender.failureCustomerID {
		sender.logger.ErrorContext(ctx, "forced notification failure",
			"service", serviceName,
			"eventId", event.EventID,
			"orderId", event.Data.OrderID,
			"eventType", event.EventType,
			"customerId", event.Data.CustomerID,
		)
		return errors.New("forced notification failure")
	}

	sender.logger.InfoContext(ctx, "notification sent",
		"service", serviceName,
		"eventId", event.EventID,
		"orderId", event.Data.OrderID,
		"eventType", event.EventType,
		"customerId", event.Data.CustomerID,
	)
	return nil
}
