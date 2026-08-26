package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

type fakeNotificationSender struct {
	events []domain.OrderCreatedEvent
	err    error
}

func (sender *fakeNotificationSender) Send(_ context.Context, event domain.OrderCreatedEvent) error {
	sender.events = append(sender.events, event)
	return sender.err
}

func TestSendNotificationHandlerHandle(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{}
	handler := NewSendNotificationHandler(sender)
	event := validHandlerOrderCreatedEvent(t)
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	err = handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-123", Body: string(body)},
	}})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(sender.events) != 1 {
		t.Fatalf("sent events = %d, want 1", len(sender.events))
	}
	if sender.events[0].EventID != event.EventID {
		t.Fatalf("sent eventId = %q, want %q", sender.events[0].EventID, event.EventID)
	}
}

func TestSendNotificationHandlerRejectsMalformedMessage(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{}
	handler := NewSendNotificationHandler(sender)

	err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-invalid", Body: `{"eventId":`},
	}})
	if err == nil {
		t.Fatal("Handle() error = nil")
	}
	if len(sender.events) != 0 {
		t.Fatal("sender was called for malformed JSON")
	}
}

func TestSendNotificationHandlerRejectsInvalidEvent(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{}
	handler := NewSendNotificationHandler(sender)
	event := validHandlerOrderCreatedEvent(t)
	event.EventType = "OrderUpdated"
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	err = handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-invalid", Body: string(body)},
	}})
	if err == nil || err.Error() != `process SQS record "message-invalid": validate OrderCreated event: eventType must be "OrderCreated"` {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(sender.events) != 0 {
		t.Fatal("sender was called for an invalid event")
	}
}

func TestSendNotificationHandlerReturnsSenderError(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{err: errors.New("provider unavailable")}
	handler := NewSendNotificationHandler(sender)
	body, err := json.Marshal(validHandlerOrderCreatedEvent(t))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	err = handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-123", Body: string(body)},
	}})
	if err == nil || err.Error() != `process SQS record "message-123": send notification: provider unavailable` {
		t.Fatalf("Handle() error = %v", err)
	}
}
