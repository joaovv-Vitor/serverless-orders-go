package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/handler"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/observability"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/orders"
)

func main() {
	logger := observability.NewJSONLogger(os.Stdout)
	startupLogger := logger.With("service", "get-order")
	tableName := os.Getenv("ORDERS_TABLE_NAME")
	if tableName == "" {
		startupLogger.Error("missing required environment variable", "name", "ORDERS_TABLE_NAME")
		os.Exit(1)
	}

	sdkConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		startupLogger.Error("failed to load AWS configuration", "error", err)
		os.Exit(1)
	}
	repository, err := orders.NewDynamoDBRepository(dynamodb.NewFromConfig(sdkConfig), tableName)
	if err != nil {
		startupLogger.Error("failed to configure orders repository", "error", err)
		os.Exit(1)
	}

	getOrderHandler := handler.NewGetOrderHandler(repository, logger)
	lambda.Start(getOrderHandler.Handle)
}
