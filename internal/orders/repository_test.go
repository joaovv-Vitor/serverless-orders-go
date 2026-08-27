package orders

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

type fakeDynamoDBClient struct {
	putInput    *dynamodb.PutItemInput
	putErr      error
	getInput    *dynamodb.GetItemInput
	getOutput   *dynamodb.GetItemOutput
	getErr      error
	updateInput *dynamodb.UpdateItemInput
	updateErr   error
}

func (client *fakeDynamoDBClient) PutItem(
	_ context.Context,
	input *dynamodb.PutItemInput,
	_ ...func(*dynamodb.Options),
) (*dynamodb.PutItemOutput, error) {
	client.putInput = input
	return &dynamodb.PutItemOutput{}, client.putErr
}

func (client *fakeDynamoDBClient) GetItem(
	_ context.Context,
	input *dynamodb.GetItemInput,
	_ ...func(*dynamodb.Options),
) (*dynamodb.GetItemOutput, error) {
	client.getInput = input
	if client.getOutput == nil {
		client.getOutput = &dynamodb.GetItemOutput{}
	}
	return client.getOutput, client.getErr
}

func (client *fakeDynamoDBClient) UpdateItem(
	_ context.Context,
	input *dynamodb.UpdateItemInput,
	_ ...func(*dynamodb.Options),
) (*dynamodb.UpdateItemOutput, error) {
	client.updateInput = input
	return &dynamodb.UpdateItemOutput{}, client.updateErr
}

func TestDynamoDBRepositoryCreate(t *testing.T) {
	t.Parallel()

	client := &fakeDynamoDBClient{}
	repository, err := NewDynamoDBRepository(client, "orders")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}
	order := testOrder()

	if err := repository.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if aws.ToString(client.putInput.TableName) != "orders" || aws.ToString(client.putInput.ConditionExpression) != "attribute_not_exists(#orderId)" {
		t.Fatalf("PutItem input = %#v", client.putInput)
	}
	var record orderRecord
	if err := attributevalue.UnmarshalMap(client.putInput.Item, &record); err != nil {
		t.Fatalf("unmarshal item: %v", err)
	}
	if record.OrderID != order.OrderID || record.Status != string(domain.OrderStatusAccepted) || len(record.Items) != 1 {
		t.Fatalf("record = %#v", record)
	}
}

func TestDynamoDBRepositoryCreateDetectsExistingOrder(t *testing.T) {
	t.Parallel()

	client := &fakeDynamoDBClient{putErr: &types.ConditionalCheckFailedException{}}
	repository, err := NewDynamoDBRepository(client, "orders")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}
	if err := repository.Create(context.Background(), testOrder()); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestDynamoDBRepositoryGet(t *testing.T) {
	t.Parallel()

	item, err := attributevalue.MarshalMap(toRecord(testOrder()))
	if err != nil {
		t.Fatalf("MarshalMap() error = %v", err)
	}
	client := &fakeDynamoDBClient{getOutput: &dynamodb.GetItemOutput{Item: item}}
	repository, err := NewDynamoDBRepository(client, "orders")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}

	order, err := repository.Get(context.Background(), "order-123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if order.OrderID != "order-123" || order.Status != domain.OrderStatusAccepted || len(order.Items) != 1 {
		t.Fatalf("order = %#v", order)
	}
	if !aws.ToBool(client.getInput.ConsistentRead) {
		t.Fatal("GetItem ConsistentRead = false")
	}
}

func TestDynamoDBRepositoryGetReturnsNotFound(t *testing.T) {
	t.Parallel()

	repository, err := NewDynamoDBRepository(&fakeDynamoDBClient{}, "orders")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}
	if _, err := repository.Get(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestDynamoDBRepositoryUpdateStatus(t *testing.T) {
	t.Parallel()

	client := &fakeDynamoDBClient{}
	repository, err := NewDynamoDBRepository(client, "orders")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}
	updatedAt := time.Date(2026, time.August, 26, 16, 0, 0, 0, time.UTC)

	if err := repository.UpdateStatus(context.Background(), "order-123", domain.OrderStatusProcessing, updatedAt); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if aws.ToString(client.updateInput.UpdateExpression) != "SET #status = :status, #updatedAt = :updatedAt" {
		t.Fatalf("UpdateExpression = %q", aws.ToString(client.updateInput.UpdateExpression))
	}
	assertStringAttribute(t, client.updateInput.Key, "orderId", "order-123")
	assertStringAttribute(t, client.updateInput.ExpressionAttributeValues, ":status", "PROCESSING")
	assertStringAttribute(t, client.updateInput.ExpressionAttributeValues, ":updatedAt", "2026-08-26T16:00:00Z")
}

func TestDynamoDBRepositoryUpdateStatusReturnsNotFound(t *testing.T) {
	t.Parallel()

	client := &fakeDynamoDBClient{updateErr: &types.ConditionalCheckFailedException{}}
	repository, err := NewDynamoDBRepository(client, "orders")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}
	if err := repository.UpdateStatus(context.Background(), "missing", domain.OrderStatusFailed, time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
}

func TestDynamoDBRepositoryReturnsClientErrors(t *testing.T) {
	t.Parallel()

	client := &fakeDynamoDBClient{
		putErr:    errors.New("put failed"),
		getErr:    errors.New("get failed"),
		updateErr: errors.New("update failed"),
	}
	repository, err := NewDynamoDBRepository(client, "orders")
	if err != nil {
		t.Fatalf("NewDynamoDBRepository() error = %v", err)
	}
	if err := repository.Create(context.Background(), testOrder()); err == nil || err.Error() != "put order: put failed" {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := repository.Get(context.Background(), "order-123"); err == nil || err.Error() != "get order: get failed" {
		t.Fatalf("Get() error = %v", err)
	}
	if err := repository.UpdateStatus(context.Background(), "order-123", domain.OrderStatusFailed, time.Now()); err == nil || err.Error() != "update order status: update failed" {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
}

func TestNewDynamoDBRepositoryValidatesConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := NewDynamoDBRepository(nil, "orders"); err == nil || err.Error() != "DynamoDB client is required" {
		t.Fatalf("NewDynamoDBRepository(nil) error = %v", err)
	}
	if _, err := NewDynamoDBRepository(&fakeDynamoDBClient{}, " "); err == nil || err.Error() != "orders table name is required" {
		t.Fatalf("NewDynamoDBRepository(blank table) error = %v", err)
	}
}

func testOrder() domain.Order {
	now := time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC)
	return domain.Order{
		OrderID:    "order-123",
		CustomerID: "customer-123",
		Items:      []domain.OrderItem{{ProductID: "product-456", Quantity: 2}},
		Status:     domain.OrderStatusAccepted,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func assertStringAttribute(t *testing.T, item map[string]types.AttributeValue, name, want string) {
	t.Helper()

	attribute, ok := item[name].(*types.AttributeValueMemberS)
	if !ok || attribute.Value != want {
		t.Fatalf("attribute %q = %#v, want %q", name, item[name], want)
	}
}
