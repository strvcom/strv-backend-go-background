package background

import (
	"context"
	"sync"
	"sync/atomic"
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
type Manager struct {
	wg               sync.WaitGroup
	len              atomic.Int32
	stalledThreshold time.Duration
	observer         Observer
	retry            Retry
}

// Options provides a means for configuring the background manager and attaching hooks to it.
type Options struct {
	// StalledThreshold is the amount of time within which the task should return before it is considered stalled. Note
	// that no effort is made to actually stop or kill the task.
	StalledThreshold time.Duration
	// Observer allow you to register monitoring functions that are called when something happens with the tasks that you
	// schedule. These are useful for logging, monitoring, etc.
	Observer Observer
	// Retry defines the default retry strategies that will be used for all tasks unless overridden by the task. Several
	// strategies are provided by github.com/kamilsk/retry/v5/strategy package.
	Retry Retry
}

// Task describes how a unit of work (a function) should be executed.
type Task struct {
	// Fn is the function to be executed in a goroutine.
	Fn Fn
	// Meta is whatever custom information you wish to associate with the task. This will be passed to the observer's
	// functions.
	Meta Metadata
	// Retry defines how the task should be retried in case of failure (if at all). This overrides the default retry
	// strategies you might have configured in the Manager. Several strategies are provided by
	// github.com/kamilsk/retry/v5/strategy package.
	Retry Retry
}

// Retry defines the functions that control the retry behavior of a task. Several strategies are provided by
// github.com/kamilsk/retry/v5/strategy package.
type Retry []strategy.Strategy

// Fn is the function to be executed in a goroutine.
type Fn func(ctx context.Context) error

// Metadata is whatever custom information you wish to associate with a task. This information will be available in your
// lifecycle hooks to help you identify which task is being processed.
type Metadata map[string]string

// NewManager creates a new instance of Manager with default options and no observer.
func NewManager() *Manager {
	return &Manager{}
}

// NewManagerWithOptions creates a new instance of Manager with the provided options and observer.
func NewManagerWithOptions(options Options) *Manager {
	return &Manager{
		stalledThreshold: options.StalledThreshold,
		observer:         options.Observer,
		retry:            options.Retry,
	}
}

// Run schedules the provided function to be executed in a goroutine.
func (m *Manager) Run(ctx context.Context, fn Fn) {
	task := Task{Fn: fn}
	m.RunTask(ctx, task)
}

// RunTask schedules the provided task to be executed in a goroutine.
func (m *Manager) RunTask(ctx context.Context, task Task) {
	m.observer.callOnTaskAdded(ctx, task)
	m.wg.Add(1)
	m.len.Add(1)

	ctx = context.WithoutCancel(ctx)
	done := make(chan error, 1)

	go m.monitor(ctx, task, done)
	go m.run(ctx, task, done)
}

// Wait blocks until all scheduled tasks have finished.
func (m *Manager) Wait() {
	m.wg.Wait()
}

// Len returns the number of currently running tasks.
func (m *Manager) Len() int32 {
	return m.len.Load()
}

func (m *Manager) run(ctx context.Context, task Task, done chan<- error) {
	strategies := mkstrategies(m.retry, task.Retry)
	done <- retry.Do(ctx, task.Fn, strategies...)
}

func (m *Manager) monitor(ctx context.Context, task Task, done <-chan error) {
	timeout := mktimeout(m.stalledThreshold)

	for {
		select {
		case <-timeout:
			m.observer.callOnTaskStalled(ctx, task)
		case err := <-done:
			if err != nil {
				m.observer.callOnTaskFailed(ctx, task, err)
			} else {
				m.observer.callOnTaskSucceeded(ctx, task)
			}

			m.wg.Done()
			m.len.Add(-1)
			return
		}
	}
}
