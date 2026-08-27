package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/handler"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/idempotency"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/observability"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/orders"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/stock"
)

func main() {
	logger := observability.NewJSONLogger(os.Stdout)
	startupLogger := logger.With("service", "process-stock")
	idempotencyTableName := os.Getenv("IDEMPOTENCY_TABLE_NAME")
	if idempotencyTableName == "" {
		startupLogger.Error("missing required environment variable", "name", "IDEMPOTENCY_TABLE_NAME")
		os.Exit(1)
	}
	ordersTableName := os.Getenv("ORDERS_TABLE_NAME")
	if ordersTableName == "" {
		startupLogger.Error("missing required environment variable", "name", "ORDERS_TABLE_NAME")
		os.Exit(1)
	}

	sdkConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		startupLogger.Error("failed to load AWS configuration", "error", err)
		os.Exit(1)
	}
	dynamoDBClient := dynamodb.NewFromConfig(sdkConfig)
	repository, err := idempotency.NewDynamoDBRepository(dynamoDBClient, idempotencyTableName)
	if err != nil {
		startupLogger.Error("failed to configure idempotency repository", "error", err)
		os.Exit(1)
	}
	orderRepository, err := orders.NewDynamoDBRepository(dynamoDBClient, ordersTableName)
	if err != nil {
		startupLogger.Error("failed to configure orders repository", "error", err)
		os.Exit(1)
	}
	guard, err := idempotency.NewGuard(repository, "process-stock")
	if err != nil {
		startupLogger.Error("failed to configure idempotency guard", "error", err)
		os.Exit(1)
	}
	processor, err := stock.NewProcessor(logger, os.Getenv("FORCE_FAILURE_CUSTOMER_ID"), guard, orderRepository)
	if err != nil {
		startupLogger.Error("failed to configure stock processor", "error", err)
		os.Exit(1)
	}

	processStockHandler := handler.NewProcessStockHandler(processor, logger)
	lambda.Start(processStockHandler.Handle)
}
