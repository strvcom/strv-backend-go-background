<div align="center">
  <h1><code>go.strv.io/background</code></h1>

  [![Continuous Integration][badge-ci]][workflow-ci] [![codecov][badge-codecov]][codecov-dashboard]

  > Never lose your goroutine again.<br />Built with ❤️ at [STRV](https://www.strv.com)
</div>

## About

This package provides mechanism to easily run a task (a function) in a goroutine and provides mechanisms to wait for all tasks to finish (a synchronisation point). Addiionally, you can attach an observer to the background manager and get notified when something interesting happens to your task - like when it finishes, errors or takes too long to complete. This allows you to centralise your error reporting and logging.

The purpose of the central synchronisation point for all your background goroutines is to make sure that your application does not exit before all goroutines have finished. You can trigger the synchronisation when your application receives a terminating signal, for example, and wait for all tasks to finish before allowing the application to exit.

## Installation

```sh
go get go.strv.io/background
```

## Usage

There are two types of tasks you can execute:

- one-off: a task that runs once and then finishes
- looping: a task that runs repeatedly until it is stopped

One-off tasks are great for triggering a single operation in the background, like sending an email or processing a file. Looping tasks are great for running a background worker that processes a queue of items, like reading from an AWS SQS queue periodically.

Additionally, each task can define its retry policies, which allows you to automatically retry the task if it fails. For one-off tasks, the task is repeated until its retry policy says it should stop, then the task is considered finished. For looping tasks, the retry policy is applied to each iteration of the task - upon failure, the task is retried using its defined retry policy and when the policy says it should stop, the task continues on to the next iteration and the process repeats.

### Initialising the manager

```go
package main

import (
  "context"

  "go.strv.io/background"
  "go.strv.io/background/observer"
)

func main() {
  manager := background.NewManagerWithOptions(background.Options{
    // Use one of the provided observers that prints logs to the console using log/slog
    // Feel free to implement your own.
    Observer: observer.Slog{},
  })

  // Share the manager with the rest of your application

  interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
  <-interrupt
  // Wait for all tasks to finish, then allow the application to exit
  manager.Close()
}
```

### Scheduling tasks

```go
import (
  "time"

  "go.strv.io/background"
  "go.strv.io/background/task"
  "github.com/kamilsk/retry/v5/strategy"
)

maanger := background.NewManagerWithOptions(background.Options{
  Observer: observer.Slog{},
})

// Executes a one-off task - the task will run only once (except if it fails and has a retry policy)
oneoff := task.Task{
  Type: task.TypeOneOff,
  Fn: func(ctx context.Context) error {
    // Do something interesting...
    <-time.After(3 * time.Second)
    return nil
  },
  Retry: task.Retry{
    strategy.Limit(3),
  }
}

manager.RunTask(context.Background(), oneoff)

// Execute a looping task - the task will run repeatedly until it is stopped
looping := task.Task{
  Type: task.TypeLoop,
  Fn: func(ctx context.Context) error {
    // Do something interesting...
    <-time.After(3 * time.Second)
    return nil
  },
  Retry: task.Retry{
    strategy.Limit(3),
  }
}

// Execute the task to be continuously run in an infinite loop until manager.Close() is called
manager.RunTask(context.Background(), looping)
```

## Examples

You can find a sample executable in the [examples](examples) folder. To run them, clone the repository and run:

```sh
go run examples/slog/slog.go
```

## License

See the [LICENSE](LICENSE) file for details.

[badge-ci]: https://github.com/strvcom/strv-backend-go-background/actions/workflows/ci.yaml/badge.svg
[workflow-ci]: https://github.com/strvcom/strv-backend-go-background/actions/workflows/ci.yaml
[badge-codecov]: https://codecov.io/gh/strvcom/strv-backend-go-background/graph/badge.svg?token=ST3JD5GCRN
[codecov-dashboard]: https://codecov.io/gh/strvcom/strv-backend-go-background
