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
		event, err := handler.processRecord(ctx, record)
		attributes := sqsLogAttributes("send-notification", record.MessageId, event)
		if err != nil {
			handler.logger.ErrorContext(ctx, "SQS record failed", append(attributes, "error", err)...)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
			continue
		}
		handler.logger.InfoContext(ctx, "SQS record completed", attributes...)
	}
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (handler SendNotificationHandler) processRecord(ctx context.Context, record events.SQSMessage) (domain.OrderCreatedEvent, error) {
	var event domain.OrderCreatedEvent
	if err := decodeJSON([]byte(record.Body), &event); err != nil {
		return event, fmt.Errorf("decode OrderCreated event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return event, fmt.Errorf("validate OrderCreated event: %w", err)
	}
	if err := handler.sender.Send(ctx, event); err != nil {
		return event, fmt.Errorf("send notification: %w", err)
	}
	return event, nil
}
