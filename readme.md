# `go.strv.io/background`

[![Tests][badge-tests]][workflow-tests] [![codecov][badge-codecov]][codecov-dashboard]

> A package that keeps track of goroutines and allows you to wait for them to finish when it's time to shut down your application.

## Purpose

In a Go application, there's often the need for background goroutines. These goroutines can be used for various purposes, but this library is focused on managing goroutines that are used for background tasks with nearly infinite lifetimes. These tasks are often used for things like periodic cleanup, background processing, or long-running connections.

This library makes that management process easier by combining the retrier resiliency pattern implemented by the [eapache/go-resiliency](https://github.com/eapache/go-resiliency) package with a pooler from the wonderful [sourcegraph/conc](https://github.com/sourcegraph/conc) library.

A typical example in production code might be a background task that periodically checks for new data in a queue and processes it. When the application is shutting down, it's important to wait for all these background tasks to finish before the process exits. This library provides a way to do that and a bit more on top of it.

## Installation

```sh
go get go.strv.io/background
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	"go.strv.io/background/manager"

func main() {
	ctx := context.Background()
	// Create a new manager.
	// The manager will cancel its context and all its tasks if any of the tasks returns an error.
	// The manager will return the first error encountered by any of its tasks.
	backgroundManager := manager.New(ctx, manager.WithCancelOnError(), manager.WithFirstError())
	//nolint
	backgroundManager.Run(
		func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				// Fail after 2 seconds.
				return context.DeadlineExceeded
			}
		},
	)
	backgroundManager.Run(
		func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				fmt.Printf("This won't be executed\n")
			}
		},
	)
	if err := backgroundManager.Wait(); err != nil {
		//nolint
		fmt.Printf("Error: %v\n", err) // Output: Error: context deadline exceeded
	}
)

```

## License

See the [LICENSE](LICENSE) file for details.

[badge-tests]: https://github.com/strvcom/strv-backend-go-background/actions/workflows/test.yaml/badge.svg
[workflow-tests]: https://github.com/strvcom/strv-backend-go-background/actions/workflows/test.yaml
[badge-codecov]: https://codecov.io/gh/strvcom/strv-backend-go-background/graph/badge.svg?token=ST3JD5GCRN
[codecov-dashboard]: https://codecov.io/gh/strvcom/strv-backend-go-background
