package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

type fakeStockProcessor struct {
	events          []domain.OrderCreatedEvent
	errorsByEventID map[string]error
}

func (processor *fakeStockProcessor) Process(_ context.Context, event domain.OrderCreatedEvent) error {
	processor.events = append(processor.events, event)
	return processor.errorsByEventID[event.EventID]
}

func TestProcessStockHandlerHandle(t *testing.T) {
	t.Parallel()

	processor := &fakeStockProcessor{}
	handler := NewProcessStockHandler(processor, discardLogger())
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
	if len(processor.events) != 1 || processor.events[0].EventID != event.EventID {
		t.Fatalf("processed events = %#v", processor.events)
	}
}

func TestProcessStockHandlerReportsMalformedMessage(t *testing.T) {
	t.Parallel()

	processor := &fakeStockProcessor{}
	handler := NewProcessStockHandler(processor, discardLogger())

	response, err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-invalid", Body: `{"eventId":`},
	}})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	assertOnlyBatchFailure(t, response, "message-invalid")
	if len(processor.events) != 0 {
		t.Fatal("processor was called for malformed JSON")
	}
}

func TestProcessStockHandlerReportsInvalidEvent(t *testing.T) {
	t.Parallel()

	processor := &fakeStockProcessor{}
	handler := NewProcessStockHandler(processor, discardLogger())
	event := validHandlerOrderCreatedEvent(t)
	event.EventVersion = 2

	response, err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-invalid", Body: marshalHandlerEvent(t, event)},
	}})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	assertOnlyBatchFailure(t, response, "message-invalid")
	if len(processor.events) != 0 {
		t.Fatal("processor was called for an invalid event")
	}
}

func TestProcessStockHandlerReportsProcessorError(t *testing.T) {
	t.Parallel()

	processor := &fakeStockProcessor{errorsByEventID: map[string]error{"event-123": errors.New("processing failed")}}
	handler := NewProcessStockHandler(processor, discardLogger())

	response, err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-123", Body: marshalHandlerEvent(t, validHandlerOrderCreatedEvent(t))},
	}})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	assertOnlyBatchFailure(t, response, "message-123")
}

func TestProcessStockHandlerReportsOnlyFailedBatchItem(t *testing.T) {
	t.Parallel()

	processor := &fakeStockProcessor{errorsByEventID: map[string]error{"event-C": errors.New("processing failed")}}
	handler := NewProcessStockHandler(processor, discardLogger())
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
	if len(processor.events) != 4 {
		t.Fatalf("processed events = %d, want all 4 attempted", len(processor.events))
	}
}

func validHandlerOrderCreatedEvent(t *testing.T) domain.OrderCreatedEvent {
	t.Helper()

	event, err := domain.NewOrderCreatedEvent(
		"event-123",
		time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC),
		"order-123",
		domain.CreateOrderInput{
			CustomerID: "customer-123",
			Items:      []domain.OrderItem{{ProductID: "product-456", Quantity: 2}},
		},
	)
	if err != nil {
		t.Fatalf("NewOrderCreatedEvent() error = %v", err)
	}
	return event
}

func marshalHandlerEvent(t *testing.T, event domain.OrderCreatedEvent) string {
	t.Helper()

	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(body)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func assertOnlyBatchFailure(t *testing.T, response events.SQSEventResponse, messageID string) {
	t.Helper()

	if len(response.BatchItemFailures) != 1 || response.BatchItemFailures[0].ItemIdentifier != messageID {
		t.Fatalf("batch failures = %#v, want only %q", response.BatchItemFailures, messageID)
	}
}
