BUILD_ARGS=go build ./cmd/idg

test:
	go test -v

build:
	$(BUILD_ARGS)

build-linux:
	env GOOS=linux GOARCH=amd64 $(BUILD_ARGS)