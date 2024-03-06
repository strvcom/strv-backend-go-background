package background

import (
	"context"
	"time"

	"github.com/kamilsk/retry/v5"
	"github.com/kamilsk/retry/v5/strategy"
)

// Manager keeps track of scheduled goroutines and provides mechanisms to wait for them to finish or cancel their
// execution. `Meta` is whatever you wish to associate with this task, usually something that will help you keep track
// of the tasks in the observer.
type Manager struct {
	stalledThreshold time.Duration
	observer         Observer
	retry            Retry
	taskmgr          taskmgr
	loopmgr          loopmgr
}

// Options provides a means for configuring the background manager and providing the observer to it.
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

// Retry defines the functions that control the retry behavior of a task. Several strategies are provided by
// github.com/kamilsk/retry/v5/strategy package.
type Retry []strategy.Strategy

// NewManager creates a new instance of Manager with default options and no observer.
func NewManager() *Manager {
	return NewManagerWithOptions(Options{})
}

// NewManagerWithOptions creates a new instance of Manager with the provided options and observer.
func NewManagerWithOptions(options Options) *Manager {
	observer := options.Observer
	if observer == nil {
		observer = DefaultObserver{}
	}

	return &Manager{
		stalledThreshold: options.StalledThreshold,
		retry:            options.Retry,
		observer:         observer,
		loopmgr:          mkloopmgr(),
	}
}

// Run schedules the provided function to be executed once in a goroutine.
func (m *Manager) Run(ctx context.Context, fn Fn) {
	task := Task{Fn: fn}
	m.RunTask(ctx, task)
}

// RunTask schedules the provided task to be executed in a goroutine. The task will be executed according to its type.
// By default, the task will be executed only once (TaskTypeOneOff).
func (m *Manager) RunTask(ctx context.Context, task Task) {
	ctx = context.WithoutCancel(ctx)
	done := make(chan error, 1)
	m.observer.OnTaskAdded(ctx, task)

	switch task.Type {
	case TaskTypeOneOff:
		m.taskmgr.start()
		go m.observe(ctx, task, done)
		go m.run(ctx, task, done)

	case TaskTypeLoop:
		m.loopmgr.start()
		go m.loop(ctx, task, done)

	default:
		m.observer.OnTaskFailed(ctx, task, ErrUnknownTaskType)
	}
}

// Wait blocks until all scheduled one-off tasks have finished. Adding more one-off tasks will prolong the wait time.
func (m *Manager) Wait() {
	m.taskmgr.group.Wait()
}

// Cancel blocks until all loop tasks finish their current loop and stops looping further. The tasks' context is not
// cancelled. Adding a new loop task after calling Cancel() will cause the task to be ignored and not run.
func (m *Manager) Cancel() {
	m.loopmgr.cancel()
}

// CountOf returns the number of tasks of the specified type that are currently running. When the TaskType is invalid it
// returns 0.
func (m *Manager) CountOf(t TaskType) int {
	switch t {
	case TaskTypeOneOff:
		return int(m.taskmgr.count.Load())
	case TaskTypeLoop:
		return int(m.loopmgr.count.Load())
	default:
		return 0
	}
}

func (m *Manager) run(ctx context.Context, task Task, done chan<- error) {
	strategies := mkstrategies(m.retry, task.Retry)
	done <- retry.Do(ctx, task.Fn, strategies...)
}

func (m *Manager) loop(ctx context.Context, task Task, done chan error) {
	defer m.loopmgr.finish()

	for {
		if m.loopmgr.ctx.Err() != nil {
			return
		}

		m.run(ctx, task, done)
		err := <-done
		if err != nil {
			m.observer.OnTaskFailed(ctx, task, err)
		}
	}
}

func (m *Manager) observe(ctx context.Context, task Task, done <-chan error) {
	timeout := mktimeout(m.stalledThreshold)
	defer m.taskmgr.finish()

	for {
		select {
		case <-timeout:
			m.observer.OnTaskStalled(ctx, task)
		case err := <-done:
			if err != nil {
				m.observer.OnTaskFailed(ctx, task, err)
			} else {
				m.observer.OnTaskSucceeded(ctx, task)
			}

			return
		}
	}
}
