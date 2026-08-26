package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

// SNSClient is the subset of the AWS SNS client needed by this project.
type SNSClient interface {
	Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// SNSPublisher serializes integration events and publishes them to one topic.
type SNSPublisher struct {
	client   SNSClient
	topicARN string
}

// NewSNSPublisher creates a publisher scoped to a single SNS topic.
func NewSNSPublisher(client SNSClient, topicARN string) (SNSPublisher, error) {
	if client == nil {
		return SNSPublisher{}, errors.New("SNS client is required")
	}
	if strings.TrimSpace(topicARN) == "" {
		return SNSPublisher{}, errors.New("SNS topic ARN is required")
	}

	return SNSPublisher{client: client, topicARN: topicARN}, nil
}

// Publish sends a validated OrderCreated event as a JSON SNS message.
func (publisher SNSPublisher) Publish(ctx context.Context, event domain.OrderCreatedEvent) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate OrderCreated event: %w", err)
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode OrderCreated event: %w", err)
	}

	_, err = publisher.client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(publisher.topicARN),
		Message:  aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("publish OrderCreated event to SNS: %w", err)
	}

	return nil
}
