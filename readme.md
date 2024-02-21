# `go.strv.io/background`

> A package that keeps track of goroutines and allows you to wait for them to finish when it's time to shut down your application.

## Purpose

In Go, when the `main` function returns, any pending goroutines are terminated. This means that we need to keep track of them somehow so that `main` can wait for them to finish before returning. This is also useful in the context of servers - when the server receives a terminating signal from the host OS (ie. due to a new release being deployed) the application needs a way to delay the shutdown long enough for the goroutines to finish before allowing itself to be terminated.

This library makes that management process easier to manage and adds some extra functionality on top, for good measure.

> ⚠️ By no means is this a replacement for proper job queue system! The intended use case is for small, relatiely fast functions that either do the actual work or schedule a job in some kind of a queue to do that work. Since even putting a job into a queue takes some time, you can remove that time from the client's request/response cycle and make your backend respond faster.

## Installation

```sh
go get go.strv.io/background
```

## Usage

```go
package main

import (
  "context"

  "go.strv.io/background"
)

type BackgroundMetadata string

func main() {
  // Create a new background manager
  mgr := background.NewManager[BackgroundMetadata]()

  mgr.Run(context.Background(), "goroutine-1", func(ctx context.Context) error {
    // Do some work here
    return nil
  })

  // Attach some monitoring logic for logging or similar purpose
  mgr.OnTaskFailed = func(ctx context.Context, meta BackgroundMetadata, err error) {
    // Log the error, inspect the metadata, etc.
  }

  // Wait for all goroutines to finish
  // Make sure you stop your components from adding more tasks
  bg.Wait()
  // Now it's safe to terminate the process
}
```

## License

See the [LICENSE](LICENSE) file for details.
