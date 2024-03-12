package task

import (
	"context"
	"errors"
	"log/slog"

	"github.com/kamilsk/retry/v5/strategy"
)

// Type determines how the task will be executed by the manager.
type Type int

const (
	// TypeOneOff is the default task type. It will be executed only once.
	TypeOneOff Type = iota
	// TypeLoop will be executed in an infinite loop until the manager's Cancel() method is called. The task will
	// restart immediately after the previous iteration returns.
	TypeLoop
)

// LogValue implements slog.LogValuer.
func (t Type) LogValue() slog.Value {
	switch t {
	case TypeOneOff:
		return slog.StringValue("oneoff")
	case TypeLoop:
		return slog.StringValue("loop")
	default:
		return slog.StringValue("invalid")
	}
}

var (
	// ErrUnknownType is returned when the task type is not a valid value of Type.
	ErrUnknownType = errors.New("unknown task type")
)

// Task describes how a unit of work (a function) should be executed.
type Task struct {
	// Fn is the function to be executed in a goroutine.
	Fn Fn
	// Type is the type of the task. It determines how the task will be executed by the manager. Default is TaskTypeOneOff.
	Type Type
	// Meta is whatever custom information you wish to associate with the task. This will be passed to the observer's
	// functions.
	Meta Metadata
	// Retry defines how the task should be retried in case of failure (if at all). This overrides the default retry
	// strategies you might have configured in the Manager. Several strategies are provided by
	// github.com/kamilsk/retry/v5/strategy package.
	Retry []strategy.Strategy
}

// LogValue implements slog.LogValuer.
func (definition Task) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("type", definition.Type),
		slog.Any("meta", definition.Meta),
	)
}

// Fn is the function to be executed in a goroutine.
type Fn func(ctx context.Context) error

// Metadata is whatever custom information you wish to associate with a task. You can access this data in the observer's
// methods to help you identify the task or get more context about it.
type Metadata map[string]string

// LogValue implements slog.LogValuer.
func (meta Metadata) LogValue() slog.Value {
	values := make([]slog.Attr, 0, len(meta))
	for key, value := range meta {
		values = append(values, slog.String(key, value))
	}

	return slog.GroupValue(values...)
}
