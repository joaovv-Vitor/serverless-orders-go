package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/handler"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/messaging"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/observability"
)

func main() {
	logger := observability.NewJSONLogger(os.Stdout)
	startupLogger := logger.With("service", "create-order")
	topicARN := os.Getenv("ORDER_EVENTS_TOPIC_ARN")
	if topicARN == "" {
		startupLogger.Error("missing required environment variable", "name", "ORDER_EVENTS_TOPIC_ARN")
		os.Exit(1)
	}

	sdkConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		startupLogger.Error("failed to load AWS configuration", "error", err)
		os.Exit(1)
	}
	publisher, err := messaging.NewSNSPublisher(sns.NewFromConfig(sdkConfig), topicARN)
	if err != nil {
		startupLogger.Error("failed to configure SNS publisher", "error", err)
		os.Exit(1)
	}

	createOrderHandler := handler.NewCreateOrderHandler(publisher, logger)
	lambda.Start(createOrderHandler.Handle)
}
