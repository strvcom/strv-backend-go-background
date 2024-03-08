package observer

import (
	"context"
	"log/slog"

	"go.strv.io/background/task"
)

// Slog is an implementation of the Observer interface that logs the events using slog.
type Slog struct{}

func (o Slog) OnTaskAdded(ctx context.Context, definition task.Task) {
	slog.Info("task:added", "task", definition)
}

func (o Slog) OnTaskSucceeded(ctx context.Context, definition task.Task) {
	slog.Info("task:succeeded", "task", definition)
}

func (o Slog) OnTaskFailed(ctx context.Context, definition task.Task, err error) {
	slog.Error("task:failed", "task", definition, "error", err)
}

func (o Slog) OnTaskStalled(ctx context.Context, definition task.Task) {
	slog.Warn("task:stalled", "task", definition)
}
