# SQS batch processing — phase 10

## Problem

Lambda polls SQS and can invoke a function with multiple records. Without
partial batch reporting, one failed record makes the entire batch visible
again, including records that already succeeded.

For this batch:

```text
A -> success
B -> success
C -> failure
D -> success
```

returning a function-level error would make `A`, `B`, `C`, and `D` eligible for
redelivery.

## Decision

Both event source mappings use:

```yaml
BatchSize: 10
FunctionResponseTypes:
  - ReportBatchItemFailures
```

The Go handlers attempt every record, log each error, and return an
`events.SQSEventResponse`. For the example above, the response is:

```json
{
  "batchItemFailures": [
    {
      "itemIdentifier": "message-C"
    }
  ]
}
```

The handler-level error is `nil`. Returning the response and also returning an
error would cause the invocation to fail as a whole.

## Result

Lambda deletes `A`, `B`, and `D` from the source queue. Only `C` becomes visible
again after the visibility timeout. If `C` keeps failing, its receive count
eventually triggers the queue's existing redrive policy and moves it to the
branch-specific DLQ.

## Interaction with idempotency

Partial batch response reduces unnecessary redelivery, but it does not replace
idempotency. SQS Standard remains an at-least-once system and can still deliver
a successful message again. The DynamoDB reservation therefore remains active
for every consumer record.

The controlled `customer-fail` path runs before the reservation. Consequently,
the intentionally failed `C` remains retryable, while successful records retain
their idempotency entries.

## Observability note

An invocation that successfully returns `batchItemFailures` is not necessarily
a Lambda function error. Operational monitoring must also consider SQS message
age, receive counts and DLQ depth. Those metrics are addressed more broadly in
phase 11.
