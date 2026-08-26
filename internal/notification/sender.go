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
	logger *slog.Logger
}

// NewSender creates the notification sender.
func NewSender(logger *slog.Logger) (Sender, error) {
	if logger == nil {
		return Sender{}, errors.New("logger is required")
	}
	return Sender{logger: logger}, nil
}

// Send records the notification that would be delivered to the customer.
func (sender Sender) Send(ctx context.Context, event domain.OrderCreatedEvent) error {
	sender.logger.InfoContext(ctx, "notification sent",
		"service", serviceName,
		"eventId", event.EventID,
		"orderId", event.Data.OrderID,
		"eventType", event.EventType,
		"customerId", event.Data.CustomerID,
	)
	return nil
}
