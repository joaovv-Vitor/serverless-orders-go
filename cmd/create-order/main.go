package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/handler"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/messaging"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	topicARN := os.Getenv("ORDER_EVENTS_TOPIC_ARN")
	if topicARN == "" {
		logger.Error("missing required environment variable", "name", "ORDER_EVENTS_TOPIC_ARN")
		os.Exit(1)
	}

	sdkConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("failed to load AWS configuration", "error", err)
		os.Exit(1)
	}
	publisher, err := messaging.NewSNSPublisher(sns.NewFromConfig(sdkConfig), topicARN)
	if err != nil {
		logger.Error("failed to configure SNS publisher", "error", err)
		os.Exit(1)
	}

	createOrderHandler := handler.NewCreateOrderHandler(publisher)
	lambda.Start(createOrderHandler.Handle)
}
