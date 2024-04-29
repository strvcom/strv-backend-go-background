package background

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"go.strv.io/background/observer"
	"go.strv.io/background/task"

	"github.com/kamilsk/retry/v5"
	"github.com/kamilsk/retry/v5/strategy"
)

var (
	// ErrUnknownType is returned when the task type is not a valid value of Type.
	ErrUnknownType = errors.New("unknown task type")
)

// Manager keeps track of running goroutines and provides mechanisms to wait for them to finish or cancel their
// execution. `Meta` is whatever you wish to associate with this task, usually something that will help you keep track
// of the tasks in the observer.
type Manager struct {
	stalledThreshold time.Duration
	observer         observer.Observer
	retry            task.Retry
	taskmgr          taskmgr
	loopmgr          loopmgr
}

// Options provides a means for configuring the background manager and providing the observer to it.
type Options struct {
	// StalledThreshold is the amount of time within which the task should return before it is considered stalled. Note
	// that no effort is made to actually stop or kill the task.
	StalledThreshold time.Duration
	// Observer allows you to register monitoring functions that are called when something happens with the tasks that you
	// execute. These are useful for logging, monitoring, etc.
	Observer observer.Observer
	// Retry defines the default retry strategies that will be used for all tasks unless overridden by the task. Several
	// strategies are provided by https://pkg.go.dev/github.com/kamilsk/retry/v5/strategy package.
	Retry task.Retry
}

// NewManager creates a new instance of Manager with default options and no observer.
func NewManager() *Manager {
	return NewManagerWithOptions(Options{})
}

// NewManagerWithOptions creates a new instance of Manager with the provided options and observer.
func NewManagerWithOptions(options Options) *Manager {
	o := options.Observer
	if o == nil {
		o = observer.Default{}
	}

	return &Manager{
		stalledThreshold: options.StalledThreshold,
		retry:            options.Retry,
		observer:         o,
		loopmgr:          mkloopmgr(),
	}
}

// Run executes the provided function once in a goroutine.
func (m *Manager) Run(ctx context.Context, fn task.Fn) {
	definition := task.Task{Fn: fn}
	m.RunTask(ctx, definition)
}

// RunTask executes the provided task in a goroutine. The task will be executed according to its type; by default, only
// once (TaskTypeOneOff).
func (m *Manager) RunTask(ctx context.Context, definition task.Task) {
	ctx = context.WithoutCancel(ctx)
	done := make(chan error, 1)
	m.observer.OnTaskAdded(ctx, definition)

	switch definition.Type {
	case task.TypeOneOff:
		m.taskmgr.start()
		go m.observe(ctx, definition, done)
		go m.run(ctx, definition, done)

	case task.TypeLoop:
		m.loopmgr.start()
		go m.loop(ctx, definition, done)

	default:
		m.observer.OnTaskFailed(ctx, definition, ErrUnknownType)
	}
}

// Wait blocks until all running one-off tasks have finished. Adding more one-off tasks will prolong the wait time.
func (m *Manager) Wait() {
	m.taskmgr.group.Wait()
}

// Cancel blocks until all loop tasks finish their current loop and stops looping further. The tasks' context is not
// cancelled. Adding a new loop task after calling Cancel() will cause the task to be ignored and not run.
func (m *Manager) Cancel() {
	m.loopmgr.cancel()
}

// Close is a convenience method that calls Wait() and Cancel() in parallel. It blocks until all tasks have finished.
func (m *Manager) Close() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		m.Wait()
		wg.Done()
	}()
	go func() {
		m.Cancel()
		wg.Done()
	}()
	wg.Wait()
}

// CountOf returns the number of tasks of the specified type that are currently running. When the TaskType is invalid it
// returns 0.
func (m *Manager) CountOf(t task.Type) int {
	switch t {
	case task.TypeOneOff:
		return int(m.taskmgr.count.Load())
	case task.TypeLoop:
		return int(m.loopmgr.count.Load())
	default:
		return 0
	}
}

func (m *Manager) run(ctx context.Context, definition task.Task, done chan<- error) {
	strategies := mkstrategies(m.retry, definition.Retry)
	done <- retry.Do(ctx, definition.Fn, strategies...)
}

func (m *Manager) loop(ctx context.Context, definition task.Task, done chan error) {
	defer m.loopmgr.finish()

	for {
		if m.loopmgr.ctx.Err() != nil {
			return
		}

		m.run(ctx, definition, done)
		err := <-done
		if err != nil {
			m.observer.OnTaskFailed(ctx, definition, err)
		}
	}
}

func (m *Manager) observe(ctx context.Context, definition task.Task, done <-chan error) {
	timeout := mktimeout(m.stalledThreshold)
	defer m.taskmgr.finish()

	for {
		select {
		case <-timeout:
			m.observer.OnTaskStalled(ctx, definition)
		case err := <-done:
			if err != nil {
				m.observer.OnTaskFailed(ctx, definition, err)
			} else {
				m.observer.OnTaskSucceeded(ctx, definition)
			}

			return
		}
	}
}

// MARK: Internal

// taskmgr is used internally for task tracking and synchronization.
type taskmgr struct {
	group sync.WaitGroup
	count atomic.Int32
}

// start tells the taskmgr that a new task has started.
func (m *taskmgr) start() {
	m.group.Add(1)
	m.count.Add(1)
}

// finish tells the taskmgr that a task has finished.
func (m *taskmgr) finish() {
	m.group.Done()
	m.count.Add(-1)
}

// loopmgr is used internally for loop tracking and synchronization and cancellation of the loops.
type loopmgr struct {
	group    sync.WaitGroup
	count    atomic.Int32
	ctx      context.Context
	cancelfn context.CancelFunc
}

func mkloopmgr() loopmgr {
	ctx, cancelfn := context.WithCancel(context.Background())
	return loopmgr{
		ctx:      ctx,
		cancelfn: cancelfn,
	}
}

// start tells the loopmgr that a new loop has started.
func (m *loopmgr) start() {
	m.group.Add(1)
	m.count.Add(1)
}

// cancel tells the loopmgr that a loop has finished.
func (m *loopmgr) finish() {
	m.group.Done()
	m.count.Add(-1)
}

func (m *loopmgr) cancel() {
	m.cancelfn()
	m.group.Wait()
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
	result := make([]strategy.Strategy, 0, max(len(defaults), len(overrides), 1))

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
