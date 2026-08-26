# Serverless Orders

An event-driven serverless application built with Go and AWS to explore asynchronous processing, messaging, resilience and observability.

## Current scope: phase 2 — HTTP API

This phase exposes the first application use case through an API Gateway HTTP
API. `POST /orders` validates the request, generates an order ID and returns
`202 Accepted`. It deliberately does not publish events or persist orders yet.

Request:

```json
{
  "customerId": "customer-123",
  "items": [
    {
      "productId": "product-456",
      "quantity": 2
    }
  ]
}
```

Response:

```json
{
  "orderId": "generated-uuid",
  "status": "accepted"
}
```

## Requirements

- Go 1.24 or newer;
- AWS SAM CLI;
- Docker, for `sam local invoke` and `sam local start-api`;
- AWS credentials are not required for local build and invocation.

## Commands

```bash
make test          # run Go tests
make validate      # validate and lint the SAM template
make build         # compile the Lambda through AWS SAM
make local-invoke  # build and invoke the Lambda in a local container
make local-api     # serve the HTTP API locally
make clean         # remove SAM build artifacts
```

The equivalent commands requested by the project are:

```bash
go test ./...
sam validate --lint
sam build
sam local invoke CreateOrderFunction --event events/api-create-order.json
```

The expected local invocation has status code `202` and a body similar to:

```json
{"orderId":"0f85d17e-5e7d-44d4-a14a-413e739dc24b","status":"accepted"}
```

To exercise the HTTP route locally, start the API:

```bash
make local-api
```

Then send a request from another terminal:

```bash
curl --request POST http://127.0.0.1:3000/orders \
  --header 'content-type: application/json' \
  --data '{"customerId":"customer-123","items":[{"productId":"product-456","quantity":2}]}'
```

## Structure

```text
.
├── cmd/create-order/main.go          # Lambda entry point
├── events/api-create-order.json      # API Gateway v2 local event
├── internal/domain/order.go          # order data and validation
├── internal/handler/create_order.go  # HTTP adapter
├── Makefile
├── go.mod
└── template.yaml                    # AWS SAM infrastructure
```

No AWS resources are created by the commands above. Resources are created only
after an explicit deployment with `sam deploy`, which is outside phase 2.
