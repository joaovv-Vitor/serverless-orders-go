.PHONY: test validate build local-invoke local-invoke-stock local-invoke-stock-failure local-invoke-notification local-invoke-notification-failure local-api clean build-CreateOrderFunction build-ProcessStockFunction build-SendNotificationFunction

LOCAL_ENV_FILE ?= events/local-env.json

# Called by SAM's makefile builder. A static binary avoids depending on the
# developer machine's C library when the function runs on Amazon Linux 2023.
build-CreateOrderFunction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-buildvcs=false \
		-tags lambda.norpc \
		-trimpath \
		-ldflags="-s -w" \
		-o "$(ARTIFACTS_DIR)/bootstrap" \
		./cmd/create-order

build-ProcessStockFunction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-buildvcs=false \
		-tags lambda.norpc \
		-trimpath \
		-ldflags="-s -w" \
		-o "$(ARTIFACTS_DIR)/bootstrap" \
		./cmd/process-stock

build-SendNotificationFunction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-buildvcs=false \
		-tags lambda.norpc \
		-trimpath \
		-ldflags="-s -w" \
		-o "$(ARTIFACTS_DIR)/bootstrap" \
		./cmd/send-notification

test:
	go test ./...

validate:
	sam validate --lint

build:
	sam build

local-invoke: build
	sam local invoke CreateOrderFunction \
		--event events/api-create-order.json \
		--env-vars "$(LOCAL_ENV_FILE)"

local-invoke-stock: build
	sam local invoke ProcessStockFunction --event events/sqs-order-created.json

local-invoke-stock-failure: build
	sam local invoke ProcessStockFunction \
		--event events/sqs-order-created-failure.json \
		--env-vars events/forced-failure-env.json

local-invoke-notification: build
	sam local invoke SendNotificationFunction --event events/sqs-order-created.json

local-invoke-notification-failure: build
	sam local invoke SendNotificationFunction \
		--event events/sqs-order-created-failure.json \
		--env-vars events/forced-failure-env.json

local-api: build
	sam local start-api --env-vars "$(LOCAL_ENV_FILE)"

clean:
	rm -rf .aws-sam
