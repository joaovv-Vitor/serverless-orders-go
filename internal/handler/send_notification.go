package handler

import (
	"context"
	"fmt"

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
}

// NewSendNotificationHandler creates an SQS handler for notifications.
func NewSendNotificationHandler(sender NotificationSender) SendNotificationHandler {
	return SendNotificationHandler{sender: sender}
}

// Handle processes every SQS record and fails the invocation on the first error.
// Partial batch responses will be introduced in phase 10.
func (handler SendNotificationHandler) Handle(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, record := range sqsEvent.Records {
		if err := handler.processRecord(ctx, record); err != nil {
			return fmt.Errorf("process SQS record %q: %w", record.MessageId, err)
		}
	}
	return nil
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
