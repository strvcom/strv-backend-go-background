# `github.com/robertrossmann/background`

> A package that keeps track of goroutines and allows you to wait for them to finish when it's time to shut down your application.

## Purpose

In Go, when the `main` function returns, any pending goroutines are terminated. This means that we need to keep track of them somehow so that `main` can wait for them to finish before returning. This is also useful in the context of servers - when the server receives a terminating signal from the host OS (ie. due to a new release being deployed) the application needs a way to delay the shutdown long enough for the goroutines to finish before allowing itself to be terminated.

This library makes that management process easier to manage and adds some extra functionality on top, for good measure.

## Installation

```sh
go get github.com/robertrossmann/background
```

## Usage

```go
package main

import (
  "context"

  "github.com/robertrossmann/background"
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
  bg.Wait()
}
```

## License

See the [LICENSE](LICENSE) file for details.
