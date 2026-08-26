package idempotency

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const timestampFormat = time.RFC3339Nano

// Repository reserves event IDs independently for each consumer.
type Repository interface {
	Claim(context.Context, string, string) (bool, error)
	Release(context.Context, string, string) error
}

// DynamoDBClient is the subset of the DynamoDB client required by the repository.
type DynamoDBClient interface {
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

// DynamoDBRepository stores idempotency reservations in DynamoDB.
type DynamoDBRepository struct {
	client    DynamoDBClient
	tableName string
	now       func() time.Time
}

// NewDynamoDBRepository creates a repository scoped to one DynamoDB table.
func NewDynamoDBRepository(client DynamoDBClient, tableName string) (DynamoDBRepository, error) {
	if client == nil {
		return DynamoDBRepository{}, errors.New("DynamoDB client is required")
	}
	if strings.TrimSpace(tableName) == "" {
		return DynamoDBRepository{}, errors.New("DynamoDB table name is required")
	}

	return DynamoDBRepository{client: client, tableName: tableName, now: time.Now}, nil
}

// Claim atomically reserves an event for one consumer.
// A conditional-check failure means that consumer has already seen the event.
func (repository DynamoDBRepository) Claim(ctx context.Context, consumer, eventID string) (bool, error) {
	_, err := repository.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(repository.tableName),
		Item: map[string]types.AttributeValue{
			"consumer":  &types.AttributeValueMemberS{Value: consumer},
			"eventId":   &types.AttributeValueMemberS{Value: eventID},
			"claimedAt": &types.AttributeValueMemberS{Value: repository.now().UTC().Format(timestampFormat)},
		},
		ConditionExpression: aws.String("attribute_not_exists(#consumer) AND attribute_not_exists(#eventId)"),
		ExpressionAttributeNames: map[string]string{
			"#consumer": "consumer",
			"#eventId":  "eventId",
		},
	})
	if err == nil {
		return true, nil
	}

	var conditionalCheckFailed *types.ConditionalCheckFailedException
	if errors.As(err, &conditionalCheckFailed) {
		return false, nil
	}
	return false, fmt.Errorf("claim idempotency key: %w", err)
}

// Release removes a reservation after a known processing failure so SQS can retry it.
func (repository DynamoDBRepository) Release(ctx context.Context, consumer, eventID string) error {
	_, err := repository.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(repository.tableName),
		Key: map[string]types.AttributeValue{
			"consumer": &types.AttributeValueMemberS{Value: consumer},
			"eventId":  &types.AttributeValueMemberS{Value: eventID},
		},
	})
	if err != nil {
		return fmt.Errorf("release idempotency key: %w", err)
	}
	return nil
}
