package manager

import (
	"context"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/sourcegraph/conc/pool"
)

// Manager keeps track of scheduled goroutines and provides mechanisms to wait for them to finish.
type Manager struct {
	pool *pool.ContextPool
	stat ManagerStats

	// retrier is a retrier instance that will be used to run tasks if retry is enabled.
	retrier *retrier.Retrier

	// cancelFunc is the cancel function of the manager's context.
	// It is called when the manager's Stop method is called.
	cancelFunc context.CancelFunc
}

// ManagerStats contains statistics about the tasks operated by the manager.
type ManagerStats struct {
	RunningTasks int `json:"running_tasks"`
}

// New creates a new instance of Manager with the provided generic type for the metadata argument.
func New(ctx context.Context, opts ...Option) *Manager {
	ctx, cancelFunc := context.WithCancel(ctx)

	p := pool.New().WithContext(ctx)
	m := &Manager{
		pool:       p,
		stat:       ManagerStats{},
		cancelFunc: cancelFunc,
	}
	for _, opt := range opts {
		opt.Apply(m)
	}
	return m
}

// TaskFunc is the function to be executed in a goroutine.
type TaskFunc func(ctx context.Context) error

// Run submits a task to the pool.
// If all workers are busy, Run will block until a worker is available.
func (m *Manager) Run(task TaskFunc) {
	taskFunc := m.withStats(m.withRetry(task, m.retrier))
	m.pool.Go(taskFunc)
}

// RunWithRetry runs a task with a dedicated retrier.
// See [Run] for more details.
func (m *Manager) RunWithRetry(task TaskFunc, retry *retrier.Retrier) {
	taskFunc := m.withStats(m.withRetry(task, retry))
	m.pool.Go(taskFunc)
}

// Stop cancels the context of the manager and all its tasks.
func (m *Manager) Stop() error {
	m.cancelFunc()
	return m.pool.Wait()
}

// Wait blocks until all scheduled tasks have finished and propagate any panics spawned by a child to the caller.
// Wait returns an error if any of the tasks failed.
func (m *Manager) Wait() error {
	defer m.cancelFunc()
	return m.pool.Wait()
}

// Stat returns manager statistics.
func (m *Manager) Stat() ManagerStats {
	return m.stat
}

func (m *Manager) withRetry(task TaskFunc, retry *retrier.Retrier) TaskFunc {
	if retry == nil {
		return task
	}
	return func(ctx context.Context) error {
		return retry.RunCtx(ctx, task)
	}
}

func (m *Manager) withStats(task TaskFunc) TaskFunc {
	return func(ctx context.Context) error {
		m.stat.RunningTasks++
		defer func() {
			m.stat.RunningTasks--
		}()
		return task(ctx)
	}
}
