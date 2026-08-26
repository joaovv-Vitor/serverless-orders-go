# Idempotency — phase 9

## Why it is needed

Amazon SQS Standard provides at-least-once delivery. A message can therefore be
delivered more than once, even when the first processing attempt succeeded. A
consumer must not assume exactly-once delivery.

## Data model

The `ProcessedEventsTable` DynamoDB table uses a composite primary key:

```text
partition key: consumer
sort key:      eventId
```

Each item also stores `claimedAt` as an RFC 3339 timestamp. Example:

```json
{
  "consumer": "process-stock",
  "eventId": "event-123",
  "claimedAt": "2026-08-26T15:30:00Z"
}
```

Including the consumer in the key is essential for fan-out. `ProcessStock` and
`SendNotification` must each process the same event once; a table keyed only by
`eventId` would let the first branch incorrectly suppress the second.

## Atomic reservation

Before performing its simulated action, each consumer calls DynamoDB `PutItem`
with this condition:

```text
attribute_not_exists(consumer) AND attribute_not_exists(eventId)
```

Only one concurrent delivery can create a given `(consumer, eventId)` item. A
`ConditionalCheckFailedException` means the item already exists, so the delivery
is logged as `duplicate event ignored` and acknowledged successfully.

A normal DynamoDB error is returned to Lambda. The SQS message is then retried;
it is not mistaken for a duplicate.

## Failures

If an operation returns a known error after acquiring the reservation, the
guard deletes the item before returning the error. This lets the next SQS
delivery try again. The controlled failures from phase 8 happen before the
reservation, so they remain suitable for demonstrating retries and DLQs.

This educational implementation has one explicit limitation: an abrupt runtime
termination immediately after the conditional write can leave a reservation
without completing the operation. A production design would normally add
states such as `IN_PROGRESS` and `COMPLETED`, plus an expiring lease or TTL. That
extra recovery protocol is intentionally outside this phase.

## Validation

Publish the exact same `OrderCreated` body twice. The expected table contents
are two items, not one:

```text
(process-stock, event-123)
(send-notification, event-123)
```

Each consumer emits its business success log once and `duplicate event ignored`
once. Unit tests exercise this behavior without requiring AWS credentials.
