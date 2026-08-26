package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/handler"
)

func main() {
	createOrderHandler := handler.NewCreateOrderHandler()
	lambda.Start(createOrderHandler.Handle)
}
