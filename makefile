all: lint test

lint: force
	./tools/golangci-lint run --verbose

test: force
	go test -v -timeout 5s -covermode=atomic -coverprofile=coverage.txt ./...

clean:
	rm -rf .cache
	rm coverage.txt

.PHONY: force
