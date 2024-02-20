all: lint test

lint: force
	./tools/golangci-lint run -v

test: force
	go test -v -timeout 5s ./...

clean:
	rm -rf .cache

.PHONY: force
