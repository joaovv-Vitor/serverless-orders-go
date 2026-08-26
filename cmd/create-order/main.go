package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/joaovv-Vitor/serverless-orders-go/internal/handler"
)

func main() {
	lambda.Start(handler.HandleBootstrap)
}
