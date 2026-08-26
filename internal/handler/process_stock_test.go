package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

type fakeStockProcessor struct {
	events []domain.OrderCreatedEvent
	err    error
}

func (processor *fakeStockProcessor) Process(_ context.Context, event domain.OrderCreatedEvent) error {
	processor.events = append(processor.events, event)
	return processor.err
}

func TestProcessStockHandlerHandle(t *testing.T) {
	t.Parallel()

	processor := &fakeStockProcessor{}
	handler := NewProcessStockHandler(processor)
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
	if len(processor.events) != 1 {
		t.Fatalf("processed events = %d, want 1", len(processor.events))
	}
	if processor.events[0].EventID != event.EventID {
		t.Fatalf("processed eventId = %q, want %q", processor.events[0].EventID, event.EventID)
	}
}

func TestProcessStockHandlerRejectsMalformedMessage(t *testing.T) {
	t.Parallel()

	processor := &fakeStockProcessor{}
	handler := NewProcessStockHandler(processor)

	err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-invalid", Body: `{"eventId":`},
	}})
	if err == nil {
		t.Fatal("Handle() error = nil")
	}
	if len(processor.events) != 0 {
		t.Fatal("processor was called for malformed JSON")
	}
}

func TestProcessStockHandlerRejectsInvalidEvent(t *testing.T) {
	t.Parallel()

	processor := &fakeStockProcessor{}
	handler := NewProcessStockHandler(processor)
	event := validHandlerOrderCreatedEvent(t)
	event.EventVersion = 2
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	err = handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-invalid", Body: string(body)},
	}})
	if err == nil || err.Error() != `process SQS record "message-invalid": validate OrderCreated event: eventVersion must be 1` {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(processor.events) != 0 {
		t.Fatal("processor was called for an invalid event")
	}
}

func TestProcessStockHandlerReturnsProcessorError(t *testing.T) {
	t.Parallel()

	processor := &fakeStockProcessor{err: errors.New("processing failed")}
	handler := NewProcessStockHandler(processor)
	body, err := json.Marshal(validHandlerOrderCreatedEvent(t))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	err = handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-123", Body: string(body)},
	}})
	if err == nil || err.Error() != `process SQS record "message-123": process stock: processing failed` {
		t.Fatalf("Handle() error = %v", err)
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
