package main

import (
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/handler"
	"github.com/joaovv-Vitor/serverless-orders-go/internal/stock"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	processor, err := stock.NewProcessor(logger)
	if err != nil {
		logger.Error("failed to configure stock processor", "error", err)
		os.Exit(1)
	}

	processStockHandler := handler.NewProcessStockHandler(processor)
	lambda.Start(processStockHandler.Handle)
}
