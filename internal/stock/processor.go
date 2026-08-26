package stock

import (
	"context"
	"errors"
	"log/slog"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/idempotency"
)

const serviceName = "process-stock"

// Processor simulates stock processing through structured logs.
type Processor struct {
	logger            *slog.Logger
	failureCustomerID string
	idempotency       idempotency.Executor
}

// NewProcessor creates the stock processor.
func NewProcessor(logger *slog.Logger, failureCustomerID string, executor idempotency.Executor) (Processor, error) {
	if logger == nil {
		return Processor{}, errors.New("logger is required")
	}
	if executor == nil {
		return Processor{}, errors.New("idempotency executor is required")
	}
	return Processor{logger: logger, failureCustomerID: failureCustomerID, idempotency: executor}, nil
}

// Process logs each item and completes without changing real stock.
func (processor Processor) Process(ctx context.Context, event domain.OrderCreatedEvent) error {
	if processor.failureCustomerID != "" && event.Data.CustomerID == processor.failureCustomerID {
		processor.logger.ErrorContext(ctx, "forced stock failure",
			"service", serviceName,
			"eventId", event.EventID,
			"orderId", event.Data.OrderID,
			"eventType", event.EventType,
			"customerId", event.Data.CustomerID,
		)
		return errors.New("forced stock failure")
	}

	executed, err := processor.idempotency.Run(ctx, event.EventID, func() error {
		return processor.process(ctx, event)
	})
	if err != nil {
		return err
	}
	if !executed {
		processor.logger.InfoContext(ctx, "duplicate event ignored",
			"service", serviceName,
			"eventId", event.EventID,
			"orderId", event.Data.OrderID,
			"eventType", event.EventType,
		)
	}
	return nil
}

func (processor Processor) process(ctx context.Context, event domain.OrderCreatedEvent) error {
	for _, item := range event.Data.Items {
		processor.logger.InfoContext(ctx, "processing stock",
			"service", serviceName,
			"eventId", event.EventID,
			"orderId", event.Data.OrderID,
			"eventType", event.EventType,
			"productId", item.ProductID,
			"quantity", item.Quantity,
		)
	}

	processor.logger.InfoContext(ctx, "stock processed",
		"service", serviceName,
		"eventId", event.EventID,
		"orderId", event.Data.OrderID,
		"eventType", event.EventType,
	)
	return nil
}
