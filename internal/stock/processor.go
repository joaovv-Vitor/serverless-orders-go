package stock

import (
	"context"
	"errors"
	"log/slog"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

const serviceName = "process-stock"

// Processor simulates stock processing through structured logs.
type Processor struct {
	logger *slog.Logger
}

// NewProcessor creates the stock processor.
func NewProcessor(logger *slog.Logger) (Processor, error) {
	if logger == nil {
		return Processor{}, errors.New("logger is required")
	}
	return Processor{logger: logger}, nil
}

// Process logs each item and completes without changing real stock.
func (processor Processor) Process(ctx context.Context, event domain.OrderCreatedEvent) error {
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
