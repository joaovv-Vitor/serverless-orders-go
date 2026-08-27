package stock

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/idempotency"
)

const serviceName = "process-stock"

// OrderStatusUpdater persists stock-processing lifecycle changes.
type OrderStatusUpdater interface {
	UpdateStatus(context.Context, string, domain.OrderStatus, time.Time) error
}

// Processor simulates stock processing through structured logs.
type Processor struct {
	logger            *slog.Logger
	failureCustomerID string
	idempotency       idempotency.Executor
	orders            OrderStatusUpdater
	now               func() time.Time
}

// NewProcessor creates the stock processor.
func NewProcessor(
	logger *slog.Logger,
	failureCustomerID string,
	executor idempotency.Executor,
	orders OrderStatusUpdater,
) (Processor, error) {
	if logger == nil {
		return Processor{}, errors.New("logger is required")
	}
	if executor == nil {
		return Processor{}, errors.New("idempotency executor is required")
	}
	if orders == nil {
		return Processor{}, errors.New("orders repository is required")
	}
	return Processor{
		logger:            logger,
		failureCustomerID: failureCustomerID,
		idempotency:       executor,
		orders:            orders,
		now:               time.Now,
	}, nil
}

// Process logs each item and completes without changing real stock.
func (processor Processor) Process(ctx context.Context, event domain.OrderCreatedEvent) error {
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
	if err := processor.orders.UpdateStatus(ctx, event.Data.OrderID, domain.OrderStatusProcessing, processor.now()); err != nil {
		return fmt.Errorf("mark order processing: %w", err)
	}

	if processor.failureCustomerID != "" && event.Data.CustomerID == processor.failureCustomerID {
		processor.logger.ErrorContext(ctx, "forced stock failure",
			"service", serviceName,
			"eventId", event.EventID,
			"orderId", event.Data.OrderID,
			"eventType", event.EventType,
			"customerId", event.Data.CustomerID,
		)
		return processor.fail(ctx, event.Data.OrderID, errors.New("forced stock failure"))
	}

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

	if err := processor.orders.UpdateStatus(ctx, event.Data.OrderID, domain.OrderStatusProcessed, processor.now()); err != nil {
		return processor.fail(ctx, event.Data.OrderID, fmt.Errorf("mark order processed: %w", err))
	}
	processor.logger.InfoContext(ctx, "stock processed",
		"service", serviceName,
		"eventId", event.EventID,
		"orderId", event.Data.OrderID,
		"eventType", event.EventType,
	)
	return nil
}

func (processor Processor) fail(ctx context.Context, orderID string, processingErr error) error {
	if err := processor.orders.UpdateStatus(ctx, orderID, domain.OrderStatusFailed, processor.now()); err != nil {
		return errors.Join(processingErr, fmt.Errorf("mark order failed: %w", err))
	}
	return processingErr
}
