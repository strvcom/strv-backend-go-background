package observer

import (
	"context"

	"go.strv.io/background/task"
)

// Multi is an implementation of the Observer interface that calls multiple observers serially.
type Multi []Observer

func (m Multi) OnTaskAdded(ctx context.Context, definition task.Task) {
	for _, o := range m {
		o.OnTaskAdded(ctx, definition)
	}
}

func (m Multi) OnTaskSucceeded(ctx context.Context, definition task.Task) {
	for _, o := range m {
		o.OnTaskSucceeded(ctx, definition)
	}
}

func (m Multi) OnTaskFailed(ctx context.Context, definition task.Task, err error) {
	for _, o := range m {
		o.OnTaskFailed(ctx, definition, err)
	}
}

func (m Multi) OnTaskStalled(ctx context.Context, definition task.Task) {
	for _, o := range m {
		o.OnTaskStalled(ctx, definition)
	}
}
