package background

import (
	"context"
	"sync"
	"time"

	"github.com/kamilsk/retry/v5"
	"github.com/kamilsk/retry/v5/strategy"
)

// Manager keeps track of scheduled goroutines and provides mechanisms to wait for them to finish. `Meta` is whatever
// you wish to associate with this task, usually something that will help you keep track of the tasks.
//
// This is useful in context of HTTP servers, where a customer request may result in some kind of background processing
// activity that should not block the response and you schedule a goroutine to handle it. However, if your server
// receives a termination signal and you do not wait for these goroutines to finish, the goroutines will be killed
// before they can run to completion. This package is not a replacement for a proper task queue system but it is a great
// package to schedule the queue jobs without the customer waiting for that to happen while at the same time being able
// to wait for all those goroutines to finish before allowing the process to exit.
type Manager[Meta any] struct {
	wg               sync.WaitGroup
	len              int
	stalledThreshold time.Duration
	hooks            Hooks[Meta]
	strategies       Retry
}

// Options provides a means for configuring the background manager and attaching hooks to it.
type Options[Meta any] struct {
	// StalledThreshold is the amount of time within which the goroutine should return before it is considered stalled.
	// Note that no effort is made to actually kill the goroutine.
	StalledThreshold time.Duration
	// Hooks allow you to register monitoring functions that are called when something happens with the goroutines that
	// you schedule. These are useful for logging, monitoring, etc.
	Hooks Hooks[Meta]
	// DefaultRetry defines the default retry strategies that will be used for all tasks unless overridden by the task.
	// Several strategies are provided by github.com/kamilsk/retry/v5/strategy package.
	DefaultRetry Retry
}

// Hooks are a set of functions that are called when certain events happen with the goroutines that you schedule. All of
// them are optional; implement only those you need.
type Hooks[Meta any] struct {
	// OnTaskAdded is called immediately after calling Run().
	OnTaskAdded func(ctx context.Context, meta Meta)
	// OnTaskSucceeded is called immediately after Task returns with no error.
	OnTaskSucceeded func(ctx context.Context, meta Meta)
	// OnTaskFailed is called immediately after Task returns with an error.
	OnTaskFailed func(ctx context.Context, meta Meta, err error)
	// OnGoroutineStalled is called when the goroutine does not return within the StalledThreshold. You can use this to
	// make sure your goroutines do not take excessive amounts of time to run to completion.
	OnGoroutineStalled func(ctx context.Context, meta Meta)
}

// TaskDefinition describes how a unit of work (a Task) should be executed.
type TaskDefinition[Meta any] struct {
	// Task is the function to be executed in a goroutine.
	Task Task
	// Meta is whatever you wish to associate with the task.
	Meta Meta
	// Retry defines how the task should be retried in case of failure (if at all). This overrides the default retry
	// strategies you might have configured in the Manager. Several strategies are provided by
	// github.com/kamilsk/retry/v5/strategy package.
	Retry Retry
}

// Retry defines the functions that control the retry behavior of a task. Several strategies are provided by
// github.com/kamilsk/retry/v5/strategy package.
type Retry []strategy.Strategy

// Task is the function to be executed in a goroutine
type Task func(ctx context.Context) error

// NilMeta is a type that can be used as Meta generic type when you do not need to associate any metadata with the task.
type NilMeta *struct{}

// NewManager creates a new instance of Manager with default options and no hooks.
func NewManager() *Manager[NilMeta] {
	return &Manager[NilMeta]{}
}

// NewManagerWithOptions creates a new instance of Manager with the provided options and hooks.
func NewManagerWithOptions[Meta any](options Options[Meta]) *Manager[Meta] {
	return &Manager[Meta]{
		stalledThreshold: options.StalledThreshold,
		hooks:            options.Hooks,
		strategies:       options.DefaultRetry,
	}
}

// Run schedules the provided task to be executed in a goroutine.
func (m *Manager[Meta]) Run(ctx context.Context, task Task) {
	definition := TaskDefinition[Meta]{Task: task}
	m.RunTaskDefinition(ctx, definition)
}

// RunTaskDefinition schedules the provided task definition to be executed in a goroutine.
func (m *Manager[Meta]) RunTaskDefinition(ctx context.Context, definition TaskDefinition[Meta]) {
	m.callOnTaskAdded(ctx, definition.Meta)
	m.wg.Add(1)
	m.len++

	ctx = context.WithoutCancel(ctx)
	done := make(chan bool, 1)

	go m.run(ctx, definition, done)
	go m.ticktock(ctx, definition.Meta, done)
}

// Wait blocks until all scheduled tasks have finished.
func (m *Manager[Meta]) Wait() {
	m.wg.Wait()
}

// Len returns the number of currently running tasks.
func (m *Manager[Meta]) Len() int {
	return m.len
}

func (m *Manager[Meta]) run(ctx context.Context, definition TaskDefinition[Meta], done chan<- bool) {
	strategies := mkstrategies(m.strategies, definition.Retry)
	err := retry.Do(ctx, definition.Task, strategies...)
	done <- true
	m.wg.Done()
	m.len--

	if err != nil {
		m.callOnTaskFailed(ctx, definition.Meta, err)
	} else {
		m.callOnTaskSucceeded(ctx, definition.Meta)
	}
}

func (m *Manager[Meta]) ticktock(ctx context.Context, meta Meta, done <-chan bool) {
	timeout := mktimeout(m.stalledThreshold)
	select {
	case <-done:
		return
	case <-timeout:
		m.callOnGoroutineStalled(ctx, meta)
		return
	}
}

func (m *Manager[Meta]) callOnTaskFailed(ctx context.Context, meta Meta, err error) {
	if m.hooks.OnTaskFailed != nil {
		m.hooks.OnTaskFailed(ctx, meta, err)
	}
}

func (m *Manager[Meta]) callOnTaskSucceeded(ctx context.Context, meta Meta) {
	if m.hooks.OnTaskSucceeded != nil {
		m.hooks.OnTaskSucceeded(ctx, meta)
	}
}

func (m *Manager[Meta]) callOnTaskAdded(ctx context.Context, meta Meta) {
	if m.hooks.OnTaskAdded != nil {
		m.hooks.OnTaskAdded(ctx, meta)
	}
}

func (m *Manager[Meta]) callOnGoroutineStalled(ctx context.Context, meta Meta) {
	if m.hooks.OnGoroutineStalled != nil {
		m.hooks.OnGoroutineStalled(ctx, meta)
	}
}

// mktimeout returns a channel that will receive the current time after the specified duration. If the duration is 0,
// the channel will never receive any message.
func mktimeout(duration time.Duration) <-chan time.Time {
	if duration == 0 {
		return make(<-chan time.Time)
	}
	return time.After(duration)
}

// mkstrategies prepares the retry strategies to be used for the task. If no defaults and no overrides are provided, a
// single execution attempt retry strategy is used. This is because the retry package would retry indefinitely on
// failure if no strategy is provided.
func mkstrategies(defaults []strategy.Strategy, overrides []strategy.Strategy) []strategy.Strategy {
	result := make([]strategy.Strategy, 0, max(len(defaults), len(overrides)))

	if len(overrides) > 0 {
		result = append(result, overrides...)
	} else {
		result = append(result, defaults...)
	}

	// If no retry strategies are provided we default to a single execution attempt
	if len(result) == 0 {
		result = append(result, strategy.Limit(1))
	}

	return result
}
