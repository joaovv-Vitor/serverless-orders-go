.PHONY: test validate build local-invoke clean build-CreateOrderFunction

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
	sam local invoke CreateOrderFunction --event events/bootstrap.json

clean:
	rm -rf .aws-sam
