# `go.strv.io/background`

[![Tests][badge-tests]][workflow-tests] [![codecov][badge-codecov]][codecov-dashboard]

> A package that keeps track of goroutines and allows you to wait for them to finish when it's time to shut down your application.

## Purpose

In Go, when the `main` function returns, any pending goroutines are terminated. This means that we need to keep track of them somehow so that `main` can wait for them to finish before returning. This is also useful in the context of servers - when the server receives a terminating signal from the host OS (ie. due to a new release being deployed) the application needs a way to delay the shutdown long enough for the goroutines to finish before allowing itself to be terminated.

This library makes that management process easier and adds some extra functionality on top, for good measure.

> ⚠️ By no means is this a replacement for proper job queue system! The intended use case is for small, relatively fast functions that either do the actual work or schedule a job in some kind of a queue to do that work. Since even putting a job into a queue takes some time, you can remove that time from the client's request/response cycle and make your backend respond faster.

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

	"go.strv.io/background"
)

// Define a type for the metadata that you want to associate with your tasks.
// The metadata is provided by the caller when a task is scheduled and is passed
// to the monitoring functions.
type TaskMetadata string

func main() {
	// Create a new background manager
	manager := background.NewManager[TaskMetadata]()
	// Define some monitoring functions for logging or error reporting
	manager.OnTaskAdded = func(ctx context.Context, meta TaskMetadata) {
		fmt.Println("Task added:", meta)
	}
	manager.OnTaskSucceeded = func(ctx context.Context, meta TaskMetadata) {
		fmt.Println("Task succeeded:", meta)
	}
	manager.OnTaskFailed = func(ctx context.Context, meta TaskMetadata, err error) {
		fmt.Println("Task failed:", meta, err)
	}

  // ... elsewhere in your codebase
  manager.Run(context.Background(), "goroutine-1", func(ctx context.Context) error {
		// Do some work here
		return nil
	})


	// Wait for all goroutines to finish
	// Make sure you stop your components from adding more tasks
	manager.Wait()
	// Now it's safe to terminate the process
}
```

## License

See the [LICENSE](LICENSE) file for details.

[badge-tests]: https://github.com/strvcom/strv-backend-go-background/actions/workflows/test.yaml/badge.svg
[workflow-tests]: https://github.com/strvcom/strv-backend-go-background/actions/workflows/test.yaml
[badge-codecov]: https://codecov.io/gh/strvcom/strv-backend-go-background/graph/badge.svg?token=ST3JD5GCRN
[codecov-dashboard]: https://codecov.io/gh/strvcom/strv-backend-go-background
