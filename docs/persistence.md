# Order persistence — phase 12

## Purpose

Persistence makes the asynchronous workflow observable through the API. A
client still receives `202 Accepted` from `POST /orders`, then can query
`GET /orders/{id}` to see the latest processing state.

This table is separate from `ProcessedEventsTable` because they model different
concepts:

- `OrdersTable` stores business-facing order state;
- `ProcessedEventsTable` stores consumer idempotency reservations.

## Data model

`OrdersTable` uses `orderId` as its partition key and on-demand billing. An item
contains:

```json
{
  "orderId": "order-123",
  "customerId": "customer-123",
  "items": [
    {
      "productId": "product-456",
      "quantity": 2
    }
  ],
  "status": "PROCESSED",
  "createdAt": "2026-08-26T15:30:00Z",
  "updatedAt": "2026-08-26T15:30:02Z"
}
```

`GetItem` uses a strongly consistent read so a query immediately after an
update does not intentionally return an older status.

## Lifecycle

```text
POST /orders
     |
     v
  ACCEPTED
     |
     v
OrderCreated -> ProcessStock
                  |
                  v
             PROCESSING
               /     \
              v       v
        PROCESSED    FAILED
```

`CreateOrder` owns the transition to `ACCEPTED`. `ProcessStock` owns
`PROCESSING`, `PROCESSED`, and `FAILED`. `SendNotification` deliberately does
not change the order status; notification delivery is an independent fan-out
concern and is not the definition of order processing in this educational model.

If SNS publication returns an error, `CreateOrder` attempts to mark the already
stored order as `FAILED`.

## HTTP query

Successful response:

```http
GET /orders/order-123
200 OK
```

```json
{
  "orderId": "order-123",
  "customerId": "customer-123",
  "items": [{"productId": "product-456", "quantity": 2}],
  "status": "PROCESSED",
  "createdAt": "2026-08-26T15:30:00Z",
  "updatedAt": "2026-08-26T15:30:02Z"
}
```

Unknown IDs return:

```http
404 Not Found
```

```json
{"message":"order not found"}
```

## IAM boundaries

- `CreateOrderFunction`: `PutItem` and `UpdateItem` on `OrdersTable`;
- `GetOrderFunction`: `GetItem` on `OrdersTable`;
- `ProcessStockFunction`: `UpdateItem` on `OrdersTable`;
- `SendNotificationFunction`: no access to `OrdersTable`.

## Distributed consistency limitation

Writing DynamoDB and publishing SNS are two separate operations. There is no
atomic transaction spanning both services. An abrupt runtime termination after
the order write but before SNS publication can leave an `ACCEPTED` order without
an event. Marking `FAILED` handles an explicit SNS error but cannot cover a
process crash in that gap.

A production system could use a transactional outbox or another recovery
mechanism. That additional architecture is intentionally not introduced in this
phase.
