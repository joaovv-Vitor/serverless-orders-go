package idempotency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDynamoDBClient struct {
	putInput    *dynamodb.PutItemInput
	putErr      error
	deleteInput *dynamodb.DeleteItemInput
	deleteErr   error
}

func (client *fakeDynamoDBClient) PutItem(
	_ context.Context,
	input *dynamodb.PutItemInput,
	_ ...func(*dynamodb.Options),
) (*dynamodb.PutItemOutput, error) {
	client.putInput = input
	return &dynamodb.PutItemOutput{}, client.putErr
}

func (client *fakeDynamoDBClient) DeleteItem(
	_ context.Context,
	input *dynamodb.DeleteItemInput,
	_ ...func(*dynamodb.Options),
) (*dynamodb.DeleteItemOutput, error) {
	client.deleteInput = input
	return &dynamodb.DeleteItemOutput{}, client.deleteErr
}

func TestDynamoDBRepositoryClaim(t *testing.T) {
	t.Parallel()

	client := &fakeDynamoDBClient{}
	repository, err := NewDynamoDBRepository(client, "processed-events")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}
	repository.now = func() time.Time {
		return time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC)
	}

	claimed, err := repository.Claim(context.Background(), "process-stock", "event-123")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if !claimed {
		t.Fatal("Claim() = false, want true")
	}
	if got := aws.ToString(client.putInput.TableName); got != "processed-events" {
		t.Fatalf("TableName = %q", got)
	}
	if got := aws.ToString(client.putInput.ConditionExpression); got != "attribute_not_exists(#consumer) AND attribute_not_exists(#eventId)" {
		t.Fatalf("ConditionExpression = %q", got)
	}
	assertStringAttribute(t, client.putInput.Item, "consumer", "process-stock")
	assertStringAttribute(t, client.putInput.Item, "eventId", "event-123")
	assertStringAttribute(t, client.putInput.Item, "claimedAt", "2026-08-26T15:30:00Z")
}

func TestDynamoDBRepositoryClaimDetectsDuplicate(t *testing.T) {
	t.Parallel()

	client := &fakeDynamoDBClient{putErr: &types.ConditionalCheckFailedException{}}
	repository, err := NewDynamoDBRepository(client, "processed-events")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}

	claimed, err := repository.Claim(context.Background(), "process-stock", "event-123")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if claimed {
		t.Fatal("Claim() = true, want false")
	}
}

func TestDynamoDBRepositoryReturnsClientErrors(t *testing.T) {
	t.Parallel()

	client := &fakeDynamoDBClient{putErr: errors.New("write failed"), deleteErr: errors.New("delete failed")}
	repository, err := NewDynamoDBRepository(client, "processed-events")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}

	if _, err := repository.Claim(context.Background(), "process-stock", "event-123"); err == nil || err.Error() != "claim idempotency key: write failed" {
		t.Fatalf("Claim() error = %v", err)
	}
	if err := repository.Release(context.Background(), "process-stock", "event-123"); err == nil || err.Error() != "release idempotency key: delete failed" {
		t.Fatalf("Release() error = %v", err)
	}
}

func TestDynamoDBRepositoryRelease(t *testing.T) {
	t.Parallel()

	client := &fakeDynamoDBClient{}
	repository, err := NewDynamoDBRepository(client, "processed-events")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}

	if err := repository.Release(context.Background(), "send-notification", "event-123"); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	assertStringAttribute(t, client.deleteInput.Key, "consumer", "send-notification")
	assertStringAttribute(t, client.deleteInput.Key, "eventId", "event-123")
}

func TestNewDynamoDBRepositoryValidatesConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := NewDynamoDBRepository(nil, "processed-events"); err == nil || err.Error() != "DynamoDB client is required" {
		t.Fatalf("NewDynamoDBRepository(nil) error = %v", err)
	}
	if _, err := NewDynamoDBRepository(&fakeDynamoDBClient{}, " "); err == nil || err.Error() != "DynamoDB table name is required" {
		t.Fatalf("NewDynamoDBRepository(blank table) error = %v", err)
	}
}

func assertStringAttribute(t *testing.T, item map[string]types.AttributeValue, name, want string) {
	t.Helper()

	attribute, ok := item[name].(*types.AttributeValueMemberS)
	if !ok {
		t.Fatalf("attribute %q = %#v, want string", name, item[name])
	}
	if attribute.Value != want {
		t.Fatalf("attribute %q = %q, want %q", name, attribute.Value, want)
	}
}
