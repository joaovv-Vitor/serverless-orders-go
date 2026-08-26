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
	"github.com/joaovv-Vitor/serverless-orders-go/internal/stock"
)

func main() {
	logger := observability.NewJSONLogger(os.Stdout)
	startupLogger := logger.With("service", "process-stock")
	tableName := os.Getenv("IDEMPOTENCY_TABLE_NAME")
	if tableName == "" {
		startupLogger.Error("missing required environment variable", "name", "IDEMPOTENCY_TABLE_NAME")
		os.Exit(1)
	}

	sdkConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		startupLogger.Error("failed to load AWS configuration", "error", err)
		os.Exit(1)
	}
	repository, err := idempotency.NewDynamoDBRepository(dynamodb.NewFromConfig(sdkConfig), tableName)
	if err != nil {
		startupLogger.Error("failed to configure idempotency repository", "error", err)
		os.Exit(1)
	}
	guard, err := idempotency.NewGuard(repository, "process-stock")
	if err != nil {
		startupLogger.Error("failed to configure idempotency guard", "error", err)
		os.Exit(1)
	}
	processor, err := stock.NewProcessor(logger, os.Getenv("FORCE_FAILURE_CUSTOMER_ID"), guard)
	if err != nil {
		startupLogger.Error("failed to configure stock processor", "error", err)
		os.Exit(1)
	}

	processStockHandler := handler.NewProcessStockHandler(processor, logger)
	lambda.Start(processStockHandler.Handle)
}
