# Serverless Orders

An event-driven serverless application built with Go and AWS to explore asynchronous processing, messaging, resilience and observability.

## Current scope: phase 1 — Bootstrap

This phase creates the smallest deployable foundation for the project: one Go
Lambda, an AWS SAM template, a local event and automated checks. It deliberately
does not include API Gateway, SNS, SQS or DynamoDB yet.

The bootstrap function accepts:

```json
{
  "name": "developer"
}
```

and returns:

```json
{
  "message": "hello, developer"
}
```

## Requirements

- Go 1.24 or newer;
- AWS SAM CLI;
- Docker, only for `sam local invoke`;
- AWS credentials are not required for local build and invocation.

## Commands

```bash
make test          # run Go tests
make validate      # validate and lint the SAM template
make build         # compile the Lambda through AWS SAM
make local-invoke  # build and invoke the Lambda in a local container
make clean         # remove SAM build artifacts
```

The equivalent commands requested by the project are:

```bash
go test ./...
sam validate --lint
sam build
sam local invoke CreateOrderFunction --event events/bootstrap.json
```

The expected local invocation payload is:

```json
{"message":"hello, developer"}
```

## Structure

```text
.
├── cmd/create-order/main.go       # Lambda entry point
├── events/bootstrap.json          # local invocation event
├── internal/handler/              # testable bootstrap logic
├── Makefile
├── go.mod
└── template.yaml                  # AWS SAM infrastructure
```

No AWS resources are created by the commands above. Resources are created only
after an explicit deployment with `sam deploy`, which is outside phase 1.
