package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/handler"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/idempotency"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/notification"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	tableName := os.Getenv("IDEMPOTENCY_TABLE_NAME")
	if tableName == "" {
		logger.Error("missing required environment variable", "name", "IDEMPOTENCY_TABLE_NAME")
		os.Exit(1)
	}

	sdkConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("failed to load AWS configuration", "error", err)
		os.Exit(1)
	}
	repository, err := idempotency.NewDynamoDBRepository(dynamodb.NewFromConfig(sdkConfig), tableName)
	if err != nil {
		logger.Error("failed to configure idempotency repository", "error", err)
		os.Exit(1)
	}
	guard, err := idempotency.NewGuard(repository, "send-notification")
	if err != nil {
		logger.Error("failed to configure idempotency guard", "error", err)
		os.Exit(1)
	}
	sender, err := notification.NewSender(logger, os.Getenv("FORCE_FAILURE_CUSTOMER_ID"), guard)
	if err != nil {
		logger.Error("failed to configure notification sender", "error", err)
		os.Exit(1)
	}

	sendNotificationHandler := handler.NewSendNotificationHandler(sender)
	lambda.Start(sendNotificationHandler.Handle)
}
