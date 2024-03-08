package observer

import (
	"context"

	"go.strv.io/background/task"
)

// Default is an implementation of the Observer interface that allows you to receive only the events you are interested
// in. It is also the default observer used by the Manager if none is provided.
type Default struct {
	// OnTaskAdded is called immediately after scheduling the Task for execution.
	HandleOnTaskAdded func(ctx context.Context, definition task.Task)
	// OnTaskSucceeded is called immediately after Task returns with no error.
	//
	// Ignored for task.TypeLoop.
	HandleOnTaskSucceeded func(ctx context.Context, definition task.Task)
	// OnTaskFailed is called after Task returns an error and all retry policy attempts have been exhausted.
	HandleOnTaskFailed func(ctx context.Context, definition task.Task, err error)
	// OnTaskStalled is called when the task does not return within the StalledThreshold. You can use this to make sure
	// your tasks do not take excessive amounts of time to run to completion.
	//
	// Ignored for task.TypeLoop.
	HandleOnTaskStalled func(ctx context.Context, definition task.Task)
}

func (o Default) OnTaskAdded(ctx context.Context, definition task.Task) {
	if o.HandleOnTaskAdded != nil {
		o.HandleOnTaskAdded(ctx, definition)
	}
}

func (o Default) OnTaskSucceeded(ctx context.Context, definition task.Task) {
	if o.HandleOnTaskSucceeded != nil {
		o.HandleOnTaskSucceeded(ctx, definition)
	}
}

func (o Default) OnTaskFailed(ctx context.Context, definition task.Task, err error) {
	if o.HandleOnTaskFailed != nil {
		o.HandleOnTaskFailed(ctx, definition, err)
	}
}

func (o Default) OnTaskStalled(ctx context.Context, definition task.Task) {
	if o.HandleOnTaskStalled != nil {
		o.HandleOnTaskStalled(ctx, definition)
	}
}
