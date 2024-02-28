package background

import "context"

// Observer includes a set of functions that are called when certain events happen with the tasks that you schedule. All
// of them are optional; implement only those you need.
type Observer struct {
	// OnTaskAdded is called immediately after adding the Task to the background manager.
	OnTaskAdded func(ctx context.Context, task Task)
	// OnTaskSucceeded is called immediately after Task returns with no error.
	OnTaskSucceeded func(ctx context.Context, task Task)
	// OnTaskFailed is called immediately after Task returns with an error.
	OnTaskFailed func(ctx context.Context, task Task, err error)
	// OnTaskStalled is called when the task does not return within the StalledThreshold. You can use this to make sure
	// your tasks do not take excessive amounts of time to run to completion.
	OnTaskStalled func(ctx context.Context, task Task)
}

func (h Observer) callOnTaskFailed(ctx context.Context, task Task, err error) {
	if h.OnTaskFailed != nil {
		h.OnTaskFailed(ctx, task, err)
	}
}

func (h Observer) callOnTaskSucceeded(ctx context.Context, task Task) {
	if h.OnTaskSucceeded != nil {
		h.OnTaskSucceeded(ctx, task)
	}
}

func (h Observer) callOnTaskAdded(ctx context.Context, task Task) {
	if h.OnTaskAdded != nil {
		h.OnTaskAdded(ctx, task)
	}
}

func (h Observer) callOnTaskStalled(ctx context.Context, task Task) {
	if h.OnTaskStalled != nil {
		h.OnTaskStalled(ctx, task)
	}
}
