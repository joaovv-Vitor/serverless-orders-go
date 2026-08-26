# Serverless Orders

An event-driven serverless application built with Go and AWS to explore asynchronous processing, messaging, resilience and observability.

## Current scope: phase 9 — Idempotent consumers

`POST /orders` validates the request, creates the versioned `OrderCreated`
event, and publishes it once to the standard SNS topic `order-events`. SNS sends
independent copies to the stock and notification queues, and each queue invokes
its own Go Lambda. Both operations are simulated through structured logs.

```text
                                  +-> stock-queue -> ProcessStock
Client -> HTTP API -> CreateOrder -> SNS
                                  +-> notification-queue -> SendNotification
```

Each source queue now has a dead-letter queue:

```text
stock-queue        -- repeated failure --> stock-dlq
notification-queue -- repeated failure --> notification-dlq
```

The SNS subscription uses raw message delivery. Therefore, the SQS message body
is the `OrderCreated` JSON itself instead of an additional SNS envelope. This
keeps the first consumer focused on the event contract.

`ProcessStock` decodes and validates `OrderCreated`, then writes one JSON log per
item and a completion log. Any error is returned to Lambda so that SQS does not
delete the message. The event source uses `BatchSize: 1`; batch and partial batch
response behavior is intentionally reserved for phase 10.

`SendNotification` independently decodes and validates the same event, then logs
the notification that would be sent to the customer. A failure in one queue does
not block processing in the other queue. Both event sources currently use
`BatchSize: 1`.

Both source queues use `maxReceiveCount: 3`. A message that repeatedly fails is
moved by SQS to its branch-specific DLQ. DLQ messages are retained for 14 days.
There is no automatic DLQ consumer or redrive in this phase.

Before performing its simulated action, each consumer atomically reserves the
pair `(consumer, eventId)` in DynamoDB. A repeated delivery is acknowledged
without repeating the operation and emits `duplicate event ignored`. Including
the consumer in the primary key preserves fan-out: stock and notification each
process the same event once. See `docs/idempotency.md` for the data model and its
explicit failure limitation.

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
make local-invoke-stock-failure # force a local stock failure
make local-invoke-notification # invoke SendNotification locally
make local-invoke-notification-failure # force a local notification failure
make local-api     # serve the HTTP API locally
make clean         # remove SAM build artifacts
```

The local checks that do not access AWS are:

```bash
go test ./...
sam validate --lint
sam build
```

Unit tests use fake SNS and DynamoDB clients. They verify conditional writes,
duplicate detection, reservation release, and the exact published event without
requiring credentials or creating cloud resources.

Normal local consumer invocation now uses the deployed DynamoDB table. Copy
`events/consumer-local-env.example.json` to the ignored
`events/consumer-local-env.json`, provide `ProcessedEventsTableName` from the
stack outputs, and use AWS credentials before running:

```bash
make local-invoke-stock
make local-invoke-notification
```

Expected output includes JSON log entries similar to:

```json
{"level":"INFO","msg":"processing stock","service":"process-stock","eventId":"event-123","orderId":"order-123","eventType":"OrderCreated","productId":"product-456","quantity":2}
{"level":"INFO","msg":"stock processed","service":"process-stock","eventId":"event-123","orderId":"order-123","eventType":"OrderCreated"}
{"level":"INFO","msg":"notification sent","service":"send-notification","eventId":"event-123","orderId":"order-123","eventType":"OrderCreated","customerId":"customer-123"}
```

Reusing the same `eventId` for a consumer produces:

```json
{"level":"INFO","msg":"duplicate event ignored","service":"process-stock","eventId":"event-123","orderId":"order-123","eventType":"OrderCreated"}
```

Controlled failures are disabled by default. To exercise them locally:

```bash
make local-invoke-stock-failure
make local-invoke-notification-failure
```

These commands are expected to return an error after emitting either `forced
stock failure` or `forced notification failure`. SAM local does not emulate SQS
receive counts or move messages to a DLQ.

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

To validate idempotency, copy `OrderEventsTopicArn` from the outputs and publish
the exact same event twice:

```bash
aws sns publish \
  --topic-arn YOUR_ORDER_EVENTS_TOPIC_ARN \
  --message file://events/order-created.json

aws sns publish \
  --topic-arn YOUR_ORDER_EVENTS_TOPIC_ARN \
  --message file://events/order-created.json
```

Each consumer should emit its success log once and `duplicate event ignored`
once. The table should contain one item for `process-stock` and another for
`send-notification`, both with `eventId=event-123`. Use a fresh `eventId` when
repeating this experiment against the same stack.

For a controlled stock failure, set `StockFailureCustomerId` to
`customer-fail` and leave `NotificationFailureCustomerId` empty. Reverse those
values to test the notification DLQ. Then submit:

```bash
curl --request POST https://YOUR_API_ID.execute-api.REGION.amazonaws.com/orders \
  --header 'content-type: application/json' \
  --data '{"customerId":"customer-fail","items":[{"productId":"product-456","quantity":2}]}'
```

After the selected consumer fails repeatedly, copy `StockDLQUrl` or
`NotificationDLQUrl` from the stack outputs and inspect it:

```bash
aws sqs receive-message \
  --queue-url YOUR_DLQ_URL \
  --max-number-of-messages 1 \
  --wait-time-seconds 10 \
  --attribute-names ApproximateReceiveCount
```

Reset both failure parameters to empty strings after testing. Detailed behavior
and validation boundaries are recorded in `docs/failure-handling.md`.

Use a stack name such as `serverless-orders`, confirm IAM role creation, and
copy `OrdersApiUrl` from the stack outputs. Then call:

```bash
curl --request POST https://YOUR_API_ID.execute-api.REGION.amazonaws.com/orders \
  --header 'content-type: application/json' \
  --data '{"customerId":"customer-123","items":[{"productId":"product-456","quantity":2}]}'
```

A `202 Accepted` response means the SNS `Publish` call completed successfully.
Open the CloudWatch logs for `ProcessStockFunction` and
`SendNotificationFunction`. The same `eventId` and `orderId` must appear in both
log groups. The expected result is:

```text
ProcessStock:      processing stock -> stock processed
SendNotification: notification sent
```

After successful processing, each Lambda deletes only the message from its own
queue. If one branch fails, that branch's message becomes visible again after
its visibility timeout while the other branch remains successfully processed.

## Structure

```text
.
├── cmd/create-order/main.go          # Lambda entry point
├── cmd/process-stock/main.go         # stock Lambda entry point
├── cmd/send-notification/main.go     # notification Lambda entry point
├── docs/failure-handling.md          # retries and DLQ behavior
├── docs/idempotency.md               # duplicate-delivery strategy
├── events/api-create-order.json      # API Gateway v2 local event
├── events/consumer-local-env.example.json # local DynamoDB configuration
├── events/local-env.example.json     # local SNS configuration example
├── events/order-created.json         # OrderCreated v1 example
├── events/sqs-order-created-failure.json # controlled failure event
├── events/sqs-order-created.json     # local SQS event
├── internal/domain/event.go          # versioned integration event
├── internal/domain/order.go          # order data and validation
├── internal/handler/create_order.go  # HTTP adapter
├── internal/handler/process_stock.go # SQS adapter
├── internal/handler/send_notification.go # notification SQS adapter
├── internal/idempotency/guard.go      # claim/execute/release workflow
├── internal/idempotency/repository.go # DynamoDB conditional-write adapter
├── internal/messaging/sns_publisher.go # AWS SNS adapter
├── internal/notification/sender.go   # simulated notification logic
├── internal/stock/processor.go       # simulated stock logic
├── Makefile
├── go.mod
└── template.yaml                    # AWS SAM infrastructure
```

`sam build` and the tests do not create AWS resources. An explicit `sam deploy`
creates the HTTP API, SNS topic, SQS queues, SNS subscriptions, queue
resource policies, three Lambda functions, two SQS event source mappings, one
on-demand DynamoDB table, and their generated execution roles. Each consumer
can poll only its own queue and can write/delete only idempotency items in that
table; neither consumer can publish to SNS.
