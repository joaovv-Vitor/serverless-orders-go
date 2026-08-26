# Serverless Orders

An event-driven serverless application built with Go and AWS to explore asynchronous processing, messaging, resilience and observability.

## Current scope: phase 6 — Stock processing

`POST /orders` validates the request, creates the versioned `OrderCreated`
event, and publishes it to the standard SNS topic `order-events`. The topic now
delivers every event to `stock-queue`, which invokes the Go Lambda
`ProcessStock`. Stock processing is still simulated through structured logs.

```text
Client -> HTTP API -> CreateOrder -> SNS -> stock-queue -> ProcessStock
```

The SNS subscription uses raw message delivery. Therefore, the SQS message body
is the `OrderCreated` JSON itself instead of an additional SNS envelope. This
keeps the first consumer focused on the event contract.

`ProcessStock` decodes and validates `OrderCreated`, then writes one JSON log per
item and a completion log. Any error is returned to Lambda so that SQS does not
delete the message. The event source uses `BatchSize: 1`; batch and partial batch
response behavior is intentionally reserved for phase 10.

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

Version 1 of the event contract:

```json
{
  "eventId": "event-123",
  "eventType": "OrderCreated",
  "eventVersion": 1,
  "occurredAt": "2026-08-26T15:30:00Z",
  "data": {
    "orderId": "order-123",
    "customerId": "customer-123",
    "items": [
      {
        "productId": "product-456",
        "quantity": 2
      }
    ]
  }
}
```

The HTTP request, internal order input and integration event use separate Go
types. This keeps changes to the public event contract explicit and prevents an
HTTP-only field from accidentally becoming part of an event.

## Requirements

- Go 1.24 or newer;
- AWS SAM CLI;
- Docker, for local Lambda/API execution;
- AWS credentials for deployment or for publishing to a real topic locally.

## Commands

```bash
make test          # run Go tests
make validate      # validate and lint the SAM template
make build         # compile the Lambda through AWS SAM
make local-invoke  # build and invoke the Lambda in a local container
make local-invoke-stock # invoke ProcessStock with a local SQS event
make local-api     # serve the HTTP API locally
make clean         # remove SAM build artifacts
```

The local checks that do not access AWS are:

```bash
go test ./...
sam validate --lint
sam build
```

Unit tests use a fake SNS client, so they verify the topic ARN and exact JSON
message without requiring credentials or creating cloud resources.

The stock consumer can be exercised fully offline:

```bash
make local-invoke-stock
```

Expected output includes JSON log entries similar to:

```json
{"level":"INFO","msg":"processing stock","service":"process-stock","eventId":"event-123","orderId":"order-123","eventType":"OrderCreated","productId":"product-456","quantity":2}
{"level":"INFO","msg":"stock processed","service":"process-stock","eventId":"event-123","orderId":"order-123","eventType":"OrderCreated"}
```

After deploying the stack, `POST /orders` returns status code `202` with a body
similar to:

```json
{"orderId":"0f85d17e-5e7d-44d4-a14a-413e739dc24b","status":"accepted"}
```

To invoke locally against an existing SNS topic, provide AWS credentials, copy
`events/local-env.example.json` to the ignored `events/local-env.json`, and
replace the placeholders with the real topic ARN:

```json
{
  "CreateOrderFunction": {
    "ORDER_EVENTS_TOPIC_ARN": "arn:aws:sns:REGION:ACCOUNT_ID:order-events"
  }
}
```

Then run either command:

```bash
make local-invoke
make local-api
```

For an end-to-end AWS check, deploy explicitly:

```bash
sam deploy --guided
```

Use a stack name such as `serverless-orders`, confirm IAM role creation, and
copy `OrdersApiUrl` from the stack outputs. Then call:

```bash
curl --request POST https://YOUR_API_ID.execute-api.REGION.amazonaws.com/orders \
  --header 'content-type: application/json' \
  --data '{"customerId":"customer-123","items":[{"productId":"product-456","quantity":2}]}'
```

A `202 Accepted` response means the SNS `Publish` call completed successfully.
Open the CloudWatch logs for the `ProcessStockFunction` output and look for the
same `orderId` returned by the API. The expected sequence is:

```text
processing stock
stock processed
```

After successful processing, Lambda deletes the message from `stock-queue`. If
decoding, validation, or processing fails, the invocation fails and the message
becomes visible again after the queue visibility timeout.

## Structure

```text
.
├── cmd/create-order/main.go          # Lambda entry point
├── cmd/process-stock/main.go         # stock Lambda entry point
├── events/api-create-order.json      # API Gateway v2 local event
├── events/local-env.example.json     # local SNS configuration example
├── events/order-created.json         # OrderCreated v1 example
├── events/sqs-order-created.json     # local SQS event
├── internal/domain/event.go          # versioned integration event
├── internal/domain/order.go          # order data and validation
├── internal/handler/create_order.go  # HTTP adapter
├── internal/handler/process_stock.go # SQS adapter
├── internal/messaging/sns_publisher.go # AWS SNS adapter
├── internal/stock/processor.go       # simulated stock logic
├── Makefile
├── go.mod
└── template.yaml                    # AWS SAM infrastructure
```

`sam build` and the tests do not create AWS resources. An explicit `sam deploy`
creates the Lambda, HTTP API, SNS topic, SQS queue, SNS subscription, queue
resource policy, both Lambda functions, the SQS event source mapping, and their
generated execution roles. `ProcessStock` can only receive and delete messages
and read attributes from `stock-queue`; it has no SNS or persistence access.
