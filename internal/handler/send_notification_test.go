package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

type fakeNotificationSender struct {
	events          []domain.OrderCreatedEvent
	errorsByEventID map[string]error
}

func (sender *fakeNotificationSender) Send(_ context.Context, event domain.OrderCreatedEvent) error {
	sender.events = append(sender.events, event)
	return sender.errorsByEventID[event.EventID]
}

func TestSendNotificationHandlerHandle(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{}
	handler := NewSendNotificationHandler(sender, discardLogger())
	event := validHandlerOrderCreatedEvent(t)

	response, err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-123", Body: marshalHandlerEvent(t, event)},
	}})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(response.BatchItemFailures) != 0 {
		t.Fatalf("batch failures = %#v, want none", response.BatchItemFailures)
	}
	if len(sender.events) != 1 || sender.events[0].EventID != event.EventID {
		t.Fatalf("sent events = %#v", sender.events)
	}
}

func TestSendNotificationHandlerReportsMalformedMessage(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{}
	handler := NewSendNotificationHandler(sender, discardLogger())

	response, err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-invalid", Body: `{"eventId":`},
	}})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	assertOnlyBatchFailure(t, response, "message-invalid")
	if len(sender.events) != 0 {
		t.Fatal("sender was called for malformed JSON")
	}
}

func TestSendNotificationHandlerReportsInvalidEvent(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{}
	handler := NewSendNotificationHandler(sender, discardLogger())
	event := validHandlerOrderCreatedEvent(t)
	event.EventType = "OrderUpdated"

	response, err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-invalid", Body: marshalHandlerEvent(t, event)},
	}})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	assertOnlyBatchFailure(t, response, "message-invalid")
	if len(sender.events) != 0 {
		t.Fatal("sender was called for an invalid event")
	}
}

func TestSendNotificationHandlerReportsSenderError(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{errorsByEventID: map[string]error{"event-123": errors.New("provider unavailable")}}
	handler := NewSendNotificationHandler(sender, discardLogger())

	response, err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-123", Body: marshalHandlerEvent(t, validHandlerOrderCreatedEvent(t))},
	}})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	assertOnlyBatchFailure(t, response, "message-123")
}

func TestSendNotificationHandlerReportsOnlyFailedBatchItem(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{errorsByEventID: map[string]error{"event-C": errors.New("provider unavailable")}}
	handler := NewSendNotificationHandler(sender, discardLogger())
	records := make([]events.SQSMessage, 0, 4)
	for _, id := range []string{"A", "B", "C", "D"} {
		event := validHandlerOrderCreatedEvent(t)
		event.EventID = "event-" + id
		event.Data.OrderID = "order-" + id
		records = append(records, events.SQSMessage{MessageId: "message-" + id, Body: marshalHandlerEvent(t, event)})
	}

	response, err := handler.Handle(context.Background(), events.SQSEvent{Records: records})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	assertOnlyBatchFailure(t, response, "message-C")
	if len(sender.events) != 4 {
		t.Fatalf("sent events = %d, want all 4 attempted", len(sender.events))
	}
}
