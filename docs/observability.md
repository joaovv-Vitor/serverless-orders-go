# Observability — phase 11

## Goals

Observability in this phase answers three practical questions:

1. Are the Lambda functions running successfully and within an expected time?
2. Are messages accumulating in a source queue or a dead-letter queue?
3. Can one order or event be followed through the asynchronous consumers?

No application emits custom metrics yet. The dashboard intentionally starts
with native Lambda and SQS metrics and structured CloudWatch Logs.

## Structured log contract

Every application log is JSON. The common fields are:

```text
time
level
service
message
eventId     when available
orderId     when available
eventType   when available
messageId   for SQS records
error       for failures
```

Example:

```json
{
  "time": "2026-08-26T15:30:00Z",
  "level": "INFO",
  "service": "process-stock",
  "message": "SQS record completed",
  "messageId": "message-123",
  "eventId": "event-123",
  "orderId": "order-123",
  "eventType": "OrderCreated"
}
```

The HTTP Lambda logs `order accepted` after SNS accepts the event. Each SQS
handler logs either `SQS record completed` or `SQS record failed`, which connects
the SQS `messageId` to the event and order identifiers.

## Log retention

SAM declares the four Lambda log groups with 14-day retention. This avoids
indefinite retention while keeping enough history for educational experiments.

## Dashboard

The stack creates `${AWS::StackName}-orders` with these native metrics:

- Lambda `Invocations`, `Errors`, and average `Duration`;
- SQS `ApproximateNumberOfMessagesVisible` for the source queues;
- SQS `ApproximateNumberOfMessagesNotVisible` for in-flight messages;
- SQS `ApproximateNumberOfMessagesVisible` for both DLQs.

SQS metrics are approximate. They describe queue health and trends rather than
an exact transaction ledger.

Partial batch failures are another important nuance: a Lambda invocation can
successfully return `batchItemFailures`, so the Lambda `Errors` metric may remain
zero while one SQS message is being retried. Queue age, visible messages, failed
record logs, and DLQ depth must therefore be considered together.

## CloudWatch Logs Insights queries

Select the four Lambda log groups before running these queries.

Follow one order:

```text
fields @timestamp, level, service, message, eventId, orderId, messageId
| filter orderId = "REPLACE_WITH_ORDER_ID"
| sort @timestamp asc
```

Follow one integration event:

```text
fields @timestamp, level, service, message, eventId, orderId, messageId
| filter eventId = "REPLACE_WITH_EVENT_ID"
| sort @timestamp asc
```

Inspect application errors:

```text
fields @timestamp, service, message, eventId, orderId, messageId, error
| filter level = "ERROR"
| sort @timestamp desc
| limit 100
```

Inspect partial batch failures:

```text
fields @timestamp, service, messageId, eventId, orderId, error
| filter message = "SQS record failed"
| sort @timestamp desc
```

CloudWatch Logs Insights automatically discovers top-level fields from the JSON
log events, so no custom parser or agent is required.

## Manual validation

Deploy the stack, submit an order, and copy `orderId` from the HTTP `202`
response. Open the dashboard named by `ObservabilityDashboardName`. Then select
the three Lambda log groups in Logs Insights and run the order query above.

The expected processing timeline is:

```text
create-order       order accepted
process-stock      SQS record completed
send-notification  SQS record completed
```

A later `GET /orders/{id}` adds `get-order: order retrieved` for the same
`orderId`.

Ordering between the two consumers is intentionally not guaranteed.
