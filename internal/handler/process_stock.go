package handler

import (
	"context"
	"fmt"
	"log/slog"

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
	logger    *slog.Logger
}

// NewProcessStockHandler creates an SQS handler for stock events.
func NewProcessStockHandler(processor StockProcessor, logger *slog.Logger) ProcessStockHandler {
	return ProcessStockHandler{processor: processor, logger: logger}
}

// Handle processes every record and reports only individual failures to Lambda.
func (handler ProcessStockHandler) Handle(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	failures := make([]events.SQSBatchItemFailure, 0)
	for _, record := range sqsEvent.Records {
		event, err := handler.processRecord(ctx, record)
		attributes := sqsLogAttributes("process-stock", record.MessageId, event)
		if err != nil {
			handler.logger.ErrorContext(ctx, "SQS record failed", append(attributes, "error", err)...)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
			continue
		}
		handler.logger.InfoContext(ctx, "SQS record completed", attributes...)
	}
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (handler ProcessStockHandler) processRecord(ctx context.Context, record events.SQSMessage) (domain.OrderCreatedEvent, error) {
	var event domain.OrderCreatedEvent
	if err := decodeJSON([]byte(record.Body), &event); err != nil {
		return event, fmt.Errorf("decode OrderCreated event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return event, fmt.Errorf("validate OrderCreated event: %w", err)
	}
	if err := handler.processor.Process(ctx, event); err != nil {
		return event, fmt.Errorf("process stock: %w", err)
	}
	return event, nil
}
