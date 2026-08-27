package orders

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/domain"
)

var (
	// ErrAlreadyExists indicates that an order ID is already present.
	ErrAlreadyExists = errors.New("order already exists")
	// ErrNotFound indicates that an order does not exist.
	ErrNotFound = errors.New("order not found")
)

// Repository contains the persistence operations needed by the application.
type Repository interface {
	Create(context.Context, domain.Order) error
	Get(context.Context, string) (domain.Order, error)
	UpdateStatus(context.Context, string, domain.OrderStatus, time.Time) error
}

// DynamoDBClient is the subset of DynamoDB used for order persistence.
type DynamoDBClient interface {
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

// DynamoDBRepository persists orders in one DynamoDB table.
type DynamoDBRepository struct {
	client    DynamoDBClient
	tableName string
}

type orderRecord struct {
	OrderID    string            `dynamodbav:"orderId"`
	CustomerID string            `dynamodbav:"customerId"`
	Items      []orderItemRecord `dynamodbav:"items"`
	Status     string            `dynamodbav:"status"`
	CreatedAt  string            `dynamodbav:"createdAt"`
	UpdatedAt  string            `dynamodbav:"updatedAt"`
}

type orderItemRecord struct {
	ProductID string `dynamodbav:"productId"`
	Quantity  int    `dynamodbav:"quantity"`
}

// NewDynamoDBRepository creates a repository scoped to one orders table.
func NewDynamoDBRepository(client DynamoDBClient, tableName string) (DynamoDBRepository, error) {
	if client == nil {
		return DynamoDBRepository{}, errors.New("DynamoDB client is required")
	}
	if strings.TrimSpace(tableName) == "" {
		return DynamoDBRepository{}, errors.New("orders table name is required")
	}
	return DynamoDBRepository{client: client, tableName: tableName}, nil
}

// Create inserts an order without overwriting an existing order ID.
func (repository DynamoDBRepository) Create(ctx context.Context, order domain.Order) error {
	item, err := attributevalue.MarshalMap(toRecord(order))
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}

	_, err = repository.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(repository.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(#orderId)"),
		ExpressionAttributeNames: map[string]string{
			"#orderId": "orderId",
		},
	})
	if err == nil {
		return nil
	}
	var conditionalCheckFailed *types.ConditionalCheckFailedException
	if errors.As(err, &conditionalCheckFailed) {
		return ErrAlreadyExists
	}
	return fmt.Errorf("put order: %w", err)
}

// Get loads one order by ID.
func (repository DynamoDBRepository) Get(ctx context.Context, orderID string) (domain.Order, error) {
	output, err := repository.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(repository.tableName),
		Key: map[string]types.AttributeValue{
			"orderId": &types.AttributeValueMemberS{Value: orderID},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return domain.Order{}, fmt.Errorf("get order: %w", err)
	}
	if len(output.Item) == 0 {
		return domain.Order{}, ErrNotFound
	}

	var record orderRecord
	if err := attributevalue.UnmarshalMap(output.Item, &record); err != nil {
		return domain.Order{}, fmt.Errorf("unmarshal order: %w", err)
	}
	order, err := fromRecord(record)
	if err != nil {
		return domain.Order{}, fmt.Errorf("decode stored order: %w", err)
	}
	return order, nil
}

// UpdateStatus updates the lifecycle status and modification timestamp.
func (repository DynamoDBRepository) UpdateStatus(
	ctx context.Context,
	orderID string,
	status domain.OrderStatus,
	updatedAt time.Time,
) error {
	if !status.Valid() {
		return fmt.Errorf("invalid order status %q", status)
	}

	_, err := repository.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(repository.tableName),
		Key: map[string]types.AttributeValue{
			"orderId": &types.AttributeValueMemberS{Value: orderID},
		},
		UpdateExpression:    aws.String("SET #status = :status, #updatedAt = :updatedAt"),
		ConditionExpression: aws.String("attribute_exists(#orderId)"),
		ExpressionAttributeNames: map[string]string{
			"#orderId":   "orderId",
			"#status":    "status",
			"#updatedAt": "updatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":    &types.AttributeValueMemberS{Value: string(status)},
			":updatedAt": &types.AttributeValueMemberS{Value: updatedAt.UTC().Format(time.RFC3339Nano)},
		},
	})
	if err == nil {
		return nil
	}
	var conditionalCheckFailed *types.ConditionalCheckFailedException
	if errors.As(err, &conditionalCheckFailed) {
		return ErrNotFound
	}
	return fmt.Errorf("update order status: %w", err)
}

func toRecord(order domain.Order) orderRecord {
	items := make([]orderItemRecord, len(order.Items))
	for index, item := range order.Items {
		items[index] = orderItemRecord{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	return orderRecord{
		OrderID:    order.OrderID,
		CustomerID: order.CustomerID,
		Items:      items,
		Status:     string(order.Status),
		CreatedAt:  order.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:  order.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func fromRecord(record orderRecord) (domain.Order, error) {
	createdAt, err := time.Parse(time.RFC3339Nano, record.CreatedAt)
	if err != nil {
		return domain.Order{}, fmt.Errorf("parse createdAt: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, record.UpdatedAt)
	if err != nil {
		return domain.Order{}, fmt.Errorf("parse updatedAt: %w", err)
	}
	status := domain.OrderStatus(record.Status)
	if !status.Valid() {
		return domain.Order{}, fmt.Errorf("invalid stored status %q", record.Status)
	}
	items := make([]domain.OrderItem, len(record.Items))
	for index, item := range record.Items {
		items[index] = domain.OrderItem{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	return domain.Order{
		OrderID:    record.OrderID,
		CustomerID: record.CustomerID,
		Items:      items,
		Status:     status,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}, nil
}
