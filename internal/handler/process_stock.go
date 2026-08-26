package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

// StockProcessor is the stock-processing capability required by the SQS handler.
type StockProcessor interface {
	Process(context.Context, domain.OrderCreatedEvent) error
}

// ProcessStockHandler adapts SQS records to the stock processor.
type ProcessStockHandler struct {
	processor StockProcessor
}

// NewProcessStockHandler creates an SQS handler for stock events.
func NewProcessStockHandler(processor StockProcessor) ProcessStockHandler {
	return ProcessStockHandler{processor: processor}
}

// Handle processes every SQS record and fails the invocation on the first error.
// Partial batch responses will be introduced in phase 10.
func (handler ProcessStockHandler) Handle(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, record := range sqsEvent.Records {
		if err := handler.processRecord(ctx, record); err != nil {
			return fmt.Errorf("process SQS record %q: %w", record.MessageId, err)
		}
	}
	return nil
}

func (handler ProcessStockHandler) processRecord(ctx context.Context, record events.SQSMessage) error {
	var event domain.OrderCreatedEvent
	if err := decodeJSON([]byte(record.Body), &event); err != nil {
		return fmt.Errorf("decode OrderCreated event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate OrderCreated event: %w", err)
	}
	if err := handler.processor.Process(ctx, event); err != nil {
		return fmt.Errorf("process stock: %w", err)
	}
	return nil
}
