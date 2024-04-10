//go:build ignore

package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/kamilsk/retry/v5/strategy"
	"go.strv.io/background"
	"go.strv.io/background/observer"
	"go.strv.io/background/task"
)

var (
	ErrSample = errors.New("sample error")
)

func main() {
	// Customise the default logger to output JSON
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("application starting - press Ctrl+C to terminate")

	manager := background.NewManagerWithOptions(background.Options{
		// Use the provided Slog observer to save some development time
		Observer: observer.Slog{},
		Retry: task.Retry{
			strategy.Limit(1),
		},
	})

	def := task.Task{
		Type: task.TypeLoop,
		Meta: task.Metadata{
			"source": "samples",
			"reason": "custom",
		},
		Fn: func(ctx context.Context) error {
			// Simulate a task that takes one second to complete but fails every time
			<-time.After(time.Second)
			return ErrSample
		},
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	manager.RunTask(context.Background(), def)

	// Wait for Ctrl+C to be pressed (usually this is the signal that the cloud provider sends to stop the container)
	sig := <-interrupt
	slog.Info("interrupted, closing manager...", "signal", sig)
	// Wait for all tasks to finish, then allow main to return, thus exiting the program
	manager.Close()
}
