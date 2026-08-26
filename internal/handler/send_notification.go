package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

// NotificationSender is the notification capability required by the SQS handler.
type NotificationSender interface {
	Send(context.Context, domain.OrderCreatedEvent) error
}

// SendNotificationHandler adapts SQS records to the notification sender.
type SendNotificationHandler struct {
	sender NotificationSender
	logger *slog.Logger
}

// NewSendNotificationHandler creates an SQS handler for notifications.
func NewSendNotificationHandler(sender NotificationSender, logger *slog.Logger) SendNotificationHandler {
	return SendNotificationHandler{sender: sender, logger: logger}
}

// Handle processes every record and reports only individual failures to Lambda.
func (handler SendNotificationHandler) Handle(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	failures := make([]events.SQSBatchItemFailure, 0)
	for _, record := range sqsEvent.Records {
		if err := handler.processRecord(ctx, record); err != nil {
			handler.logger.ErrorContext(ctx, "SQS record failed",
				"service", "send-notification",
				"messageId", record.MessageId,
				"error", err,
			)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (handler SendNotificationHandler) processRecord(ctx context.Context, record events.SQSMessage) error {
	var event domain.OrderCreatedEvent
	if err := decodeJSON([]byte(record.Body), &event); err != nil {
		return fmt.Errorf("decode OrderCreated event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate OrderCreated event: %w", err)
	}
	if err := handler.sender.Send(ctx, event); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	return nil
}
