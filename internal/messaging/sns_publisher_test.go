package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

type fakeSNSClient struct {
	input *sns.PublishInput
	err   error
}

func (client *fakeSNSClient) Publish(
	_ context.Context,
	input *sns.PublishInput,
	_ ...func(*sns.Options),
) (*sns.PublishOutput, error) {
	client.input = input
	if client.err != nil {
		return nil, client.err
	}
	return &sns.PublishOutput{MessageId: aws.String("message-123")}, nil
}

func TestSNSPublisherPublish(t *testing.T) {
	t.Parallel()

	client := &fakeSNSClient{}
	publisher, err := NewSNSPublisher(client, "arn:aws:sns:us-east-1:123456789012:order-events")
	if err != nil {
		t.Fatalf("NewSNSPublisher() error = %v", err)
	}
	event := validOrderCreatedEvent(t)

	if err := publisher.Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if client.input == nil {
		t.Fatal("SNS Publish was not called")
	}
	if got := aws.ToString(client.input.TopicArn); got != "arn:aws:sns:us-east-1:123456789012:order-events" {
		t.Fatalf("TopicArn = %q", got)
	}

	var published domain.OrderCreatedEvent
	if err := json.Unmarshal([]byte(aws.ToString(client.input.Message)), &published); err != nil {
		t.Fatalf("published message is not valid JSON: %v", err)
	}
	if published.EventID != event.EventID || published.Data.OrderID != event.Data.OrderID {
		t.Fatalf("published event = %#v, want %#v", published, event)
	}
}

func TestSNSPublisherPublishReturnsClientError(t *testing.T) {
	t.Parallel()

	client := &fakeSNSClient{err: errors.New("SNS unavailable")}
	publisher, err := NewSNSPublisher(client, "arn:aws:sns:us-east-1:123456789012:order-events")
	if err != nil {
		t.Fatalf("NewSNSPublisher() error = %v", err)
	}

	err = publisher.Publish(context.Background(), validOrderCreatedEvent(t))
	if err == nil || err.Error() != "publish OrderCreated event to SNS: SNS unavailable" {
		t.Fatalf("Publish() error = %v", err)
	}
}

func TestSNSPublisherRejectsInvalidEvent(t *testing.T) {
	t.Parallel()

	client := &fakeSNSClient{}
	publisher, err := NewSNSPublisher(client, "arn:aws:sns:us-east-1:123456789012:order-events")
	if err != nil {
		t.Fatalf("NewSNSPublisher() error = %v", err)
	}

	err = publisher.Publish(context.Background(), domain.OrderCreatedEvent{})
	if err == nil || err.Error() != "validate OrderCreated event: eventId is required" {
		t.Fatalf("Publish() error = %v", err)
	}
	if client.input != nil {
		t.Fatal("SNS Publish was called for an invalid event")
	}
}

func TestNewSNSPublisherValidatesConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		client  SNSClient
		topic   string
		wantErr string
	}{
		{name: "missing client", topic: "topic-arn", wantErr: "SNS client is required"},
		{name: "missing topic", client: &fakeSNSClient{}, wantErr: "SNS topic ARN is required"},
		{name: "blank topic", client: &fakeSNSClient{}, topic: "  ", wantErr: "SNS topic ARN is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewSNSPublisher(tt.client, tt.topic)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("NewSNSPublisher() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func validOrderCreatedEvent(t *testing.T) domain.OrderCreatedEvent {
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
