package background

import (
	"context"
	"errors"
)

// TaskType determines how the task will be executed by the manager.
type TaskType int

const (
	// TaskTypeOneOff is the default task type. It will be executed only once.
	TaskTypeOneOff TaskType = iota
	// TaskTypeLoop will be executed in an infinite loop until the manager's Cancel() method is called. The task will
	// restart immediately after the previous iteration returns.
	TaskTypeLoop
)

var (
	// ErrUnknownTaskType is returned when the task type is not a valid value of TaskType.
	ErrUnknownTaskType = errors.New("unknown task type")
)

// Task describes how a unit of work (a function) should be executed.
type Task struct {
	// Fn is the function to be executed in a goroutine.
	Fn Fn
	// Type is the type of the task. It determines how the task will be executed by the manager. Default is TaskTypeOneOff.
	Type TaskType
	// Meta is whatever custom information you wish to associate with the task. This will be passed to the observer's
	// functions.
	Meta Metadata
	// Retry defines how the task should be retried in case of failure (if at all). This overrides the default retry
	// strategies you might have configured in the Manager. Several strategies are provided by
	// github.com/kamilsk/retry/v5/strategy package.
	Retry Retry
}

// Fn is the function to be executed in a goroutine.
type Fn func(ctx context.Context) error

// Metadata is whatever custom information you wish to associate with a task. You can access this data in the observer's
// methods to help you identify the task or get more context about it.
type Metadata map[string]string
