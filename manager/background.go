package manager

import (
	"context"
	"sync"
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
	wg  sync.WaitGroup
	len int

	// OnTaskAdded is called immediately after calling Run(). You can use this for logging, metrics or other purposes.
	OnTaskAdded func(ctx context.Context, meta Meta)
	// OnTaskSucceeded is called immediately after Task returns. You can use this for logging, metrics or other purposes.
	OnTaskSucceeded func(ctx context.Context, meta Meta)
	// OnTaskFailed is called immediately after Task returns with an error. You can use this for logging, metrics or other
	// purposes.
	OnTaskFailed func(ctx context.Context, meta Meta, err error)
}

// Task is the function to be executed in a goroutine
type Task func(ctx context.Context) error

// NewManager creates a new instance of Manager with the provided generic type for the metadata argument.
func New[Meta any]() *Manager[Meta] {
	return &Manager[Meta]{}
}

// Run schedules the provided task to be executed in a goroutine. `Meta` is whatever you wish to associate with the
// task.
func (m *Manager[Meta]) Run(ctx context.Context, meta Meta, task Task) {
	m.callOnTaskAdded(ctx, meta)
	m.wg.Add(1)
	m.len++
	go m.run(context.WithoutCancel(ctx), meta, task)
}

// Wait blocks until all scheduled tasks have finished.
func (m *Manager[Meta]) Wait() {
	m.wg.Wait()
}

// Len returns the number of currently running tasks
func (m *Manager[Meta]) Len() int {
	return m.len
}

func (m *Manager[Meta]) run(ctx context.Context, meta Meta, task Task) {
	defer func() {
		m.wg.Done()
		m.len--
	}()

	err := task(ctx)

	if err != nil {
		m.callOnTaskFailed(ctx, meta, err)
	} else {
		m.callOnTaskSucceeded(ctx, meta)
	}
}

func (m *Manager[Meta]) callOnTaskFailed(ctx context.Context, meta Meta, err error) {
	if m.OnTaskFailed != nil {
		m.OnTaskFailed(ctx, meta, err)
	}
}

func (m *Manager[Meta]) callOnTaskSucceeded(ctx context.Context, meta Meta) {
	if m.OnTaskSucceeded != nil {
		m.OnTaskSucceeded(ctx, meta)
	}
}

func (m *Manager[Meta]) callOnTaskAdded(ctx context.Context, meta Meta) {
	if m.OnTaskAdded != nil {
		m.OnTaskAdded(ctx, meta)
	}
}
