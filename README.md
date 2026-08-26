# Serverless Orders

An event-driven serverless application built with Go and AWS to explore asynchronous processing, messaging, resilience and observability.

## Current scope: phase 4 — SNS publishing

`POST /orders` validates the request, creates the versioned `OrderCreated`
event, publishes it to the standard SNS topic `order-events`, and returns
`202 Accepted` only after SNS accepts the publication. There are no subscribers
yet; the first SQS subscription belongs to phase 5.

```text
Client -> HTTP API -> CreateOrder -> OrderCreated -> SNS order-events
```

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
The topic's CloudWatch `NumberOfMessagesPublished` metric can also be checked.
There is intentionally no subscription to receive the message in this phase.

## Structure

```text
.
├── cmd/create-order/main.go          # Lambda entry point
├── events/api-create-order.json      # API Gateway v2 local event
├── events/local-env.example.json     # local SNS configuration example
├── events/order-created.json         # OrderCreated v1 example
├── internal/domain/event.go          # versioned integration event
├── internal/domain/order.go          # order data and validation
├── internal/handler/create_order.go  # HTTP adapter
├── internal/messaging/sns_publisher.go # AWS SNS adapter
├── Makefile
├── go.mod
└── template.yaml                    # AWS SAM infrastructure
```

`sam build` and the tests do not create AWS resources. An explicit `sam deploy`
creates the Lambda, HTTP API, SNS topic and generated IAM execution role. The
role grants `sns:Publish` only for the `order-events` topic declared by this
stack. Because the topic has no subscriber in phase 4, published messages are
not retained for later consumption.
