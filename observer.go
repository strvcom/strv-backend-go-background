package background

import "context"

// Observer implements a set of methods that are called when certain events happen with the tasks that you schedule.
type Observer interface {
	// OnTaskAdded is called immediately after scheduling the Task for execution.
	OnTaskAdded(ctx context.Context, task Task)
	// OnTaskSucceeded is called immediately after Task returns with no error.
	//
	// Ignored for TaskTypeLoop.
	OnTaskSucceeded(ctx context.Context, task Task)
	// OnTaskFailed is called after Task returns an error and all retry policy attempts have been exhausted.
	OnTaskFailed(ctx context.Context, task Task, err error)
	// OnTaskStalled is called when the task does not return within the StalledThreshold. You can use this to make sure
	// your tasks do not take excessive amounts of time to run to completion.
	//
	// Ignored for TaskTypeLoop.
	OnTaskStalled(ctx context.Context, task Task)
}

// DefaultObserver is an implementation of the Observer interface that allows you to observe only the events you are
// interested in.
type DefaultObserver struct {
	// OnTaskAdded is called immediately after scheduling the Task for execution.
	HandleOnTaskAdded func(ctx context.Context, task Task)
	// OnTaskSucceeded is called immediately after Task returns with no error.
	//
	// Ignored for TaskTypeLoop.
	HandleOnTaskSucceeded func(ctx context.Context, task Task)
	// OnTaskFailed is called after Task returns an error and all retry policy attempts have been exhausted.
	HandleOnTaskFailed func(ctx context.Context, task Task, err error)
	// OnTaskStalled is called when the task does not return within the StalledThreshold. You can use this to make sure
	// your tasks do not take excessive amounts of time to run to completion.
	//
	// Ignored for TaskTypeLoop.
	HandleOnTaskStalled func(ctx context.Context, task Task)
}

func (o DefaultObserver) OnTaskFailed(ctx context.Context, task Task, err error) {
	if o.HandleOnTaskFailed != nil {
		o.HandleOnTaskFailed(ctx, task, err)
	}
}

func (o DefaultObserver) OnTaskSucceeded(ctx context.Context, task Task) {
	if o.HandleOnTaskSucceeded != nil {
		o.HandleOnTaskSucceeded(ctx, task)
	}
}

func (o DefaultObserver) OnTaskAdded(ctx context.Context, task Task) {
	if o.HandleOnTaskAdded != nil {
		o.HandleOnTaskAdded(ctx, task)
	}
}

func (o DefaultObserver) OnTaskStalled(ctx context.Context, task Task) {
	if o.HandleOnTaskStalled != nil {
		o.HandleOnTaskStalled(ctx, task)
	}
}
