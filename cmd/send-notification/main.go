package main

import (
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/handler"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/notification"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	sender, err := notification.NewSender(logger, os.Getenv("FORCE_FAILURE_CUSTOMER_ID"))
	if err != nil {
		logger.Error("failed to configure notification sender", "error", err)
		os.Exit(1)
	}

	sendNotificationHandler := handler.NewSendNotificationHandler(sender)
	lambda.Start(sendNotificationHandler.Handle)
}
