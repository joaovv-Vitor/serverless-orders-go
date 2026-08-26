package notification

import (
	"context"
	"errors"
	"log/slog"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/idempotency"
)

const serviceName = "send-notification"

// Sender simulates customer notification through a structured log.
type Sender struct {
	logger            *slog.Logger
	failureCustomerID string
	idempotency       idempotency.Executor
}

// NewSender creates the notification sender.
func NewSender(logger *slog.Logger, failureCustomerID string, executor idempotency.Executor) (Sender, error) {
	if logger == nil {
		return Sender{}, errors.New("logger is required")
	}
	if executor == nil {
		return Sender{}, errors.New("idempotency executor is required")
	}
	return Sender{logger: logger, failureCustomerID: failureCustomerID, idempotency: executor}, nil
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

	executed, err := sender.idempotency.Run(ctx, event.EventID, func() error {
		return sender.send(ctx, event)
	})
	if err != nil {
		return err
	}
	if !executed {
		sender.logger.InfoContext(ctx, "duplicate event ignored",
			"service", serviceName,
			"eventId", event.EventID,
			"orderId", event.Data.OrderID,
			"eventType", event.EventType,
		)
	}
	return nil
}

func (sender Sender) send(ctx context.Context, event domain.OrderCreatedEvent) error {
	sender.logger.InfoContext(ctx, "notification sent",
		"service", serviceName,
		"eventId", event.EventID,
		"orderId", event.Data.OrderID,
		"eventType", event.EventType,
		"customerId", event.Data.CustomerID,
	)
	return nil
}
