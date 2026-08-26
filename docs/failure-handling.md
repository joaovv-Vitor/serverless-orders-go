# Failure handling — phase 8

## Current behavior

Both SQS consumers add the failed message ID to `batchItemFailures` when JSON
decoding, event validation, or simulated processing fails. Lambda acknowledges
the successful records and leaves only each reported message unacknowledged.
SQS makes that message visible again after the visibility timeout.

The source queues use `maxReceiveCount: 3`. After repeated failed receives, SQS
moves the message to the queue-specific dead-letter queue:

```text
stock-queue        -> stock-dlq
notification-queue -> notification-dlq
```

DLQ messages are retained for 14 days. No Lambda consumes or automatically
redrives a DLQ in this phase.

## Controlled failure

Failure injection is disabled by default. Each consumer receives its own
optional `FORCE_FAILURE_CUSTOMER_ID` value from a SAM parameter:

- `StockFailureCustomerId` for `ProcessStock`;
- `NotificationFailureCustomerId` for `SendNotification`.

When the configured value matches `event.data.customerId`, that consumer logs a
structured error and returns a deterministic failure. The other fan-out branch
continues independently.

## Locally observed

The normal and forced-failure paths were invoked with SAM local. The normal
events completed and emitted their success logs. The controlled events emit
`forced stock failure` or `forced notification failure`, and the handler returns
their message IDs in `batchItemFailures`.

SAM local invokes a function once. It does not poll SQS, update a receive count,
wait for a visibility timeout, or move a message to a DLQ. Those behaviors must
be verified against a deployed stack.

## Cloud verification

Configure one failure parameter at a time, deploy, and send an order with
`customerId` equal to `customer-fail`. Observe repeated errors in the selected
consumer and a successful execution in the other branch. After the configured
receive attempts, inspect the corresponding DLQ and verify that it contains the
original `OrderCreated` body.

Reset both failure parameters to empty strings after the experiment. Messages
in a DLQ remain there until they expire, are explicitly deleted, or are manually
redriven.
