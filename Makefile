.PHONY: test validate build local-invoke local-api clean build-CreateOrderFunction

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

test:
	go test ./...

validate:
	sam validate --lint

build:
	sam build

local-invoke: build
	sam local invoke CreateOrderFunction --event events/api-create-order.json

local-api: build
	sam local start-api

clean:
	rm -rf .aws-sam
