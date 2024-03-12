package observer

import (
	"context"

	"go.strv.io/background/task"
)

// Observer implements a set of methods that are called when certain events happen with the tasks that you schedule.
type Observer interface {
	// OnTaskAdded is called immediately after scheduling the Task for execution.
	OnTaskAdded(ctx context.Context, definition task.Task)
	// OnTaskSucceeded is called immediately after Task returns with no error.
	//
	// Ignored for task.TypeLoop.
	OnTaskSucceeded(ctx context.Context, definition task.Task)
	// OnTaskFailed is called after Task returns an error and all retry policy attempts have been exhausted.
	OnTaskFailed(ctx context.Context, definition task.Task, err error)
	// OnTaskStalled is called when the task does not return within the StalledThreshold. You can use this to make sure
	// your tasks do not take excessive amounts of time to run to completion.
	//
	// Ignored for task.TypeLoop.
	OnTaskStalled(ctx context.Context, definition task.Task)
}
