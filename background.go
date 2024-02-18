package background

import (
	"context"
	"sync"
)

// Executor keeps track of scheduled goroutines and provides mechanisms to wait for them to finish. `Meta` is whatever
// you wish to associate with this task, usually something that will help you keep track of the tasks.
//
// This is useful in context of HTTP servers, where a customer request may result in some kind of background processing
// activity that should not block the response and you schedule a goroutine to handle it. However, if your server
// receives a termination signal and you do not wait for these goroutines to finish, the goroutines will be killed
// before they can run to completion. This package is not a replacement for a proper task queue system but it is a great
// package to schedule the queue jobs without the customer waiting for that to happen while at the same time being able
// to wait for all those goroutines to finish before allowing the process to exit.
type Executor[Meta any] struct {
	wg sync.WaitGroup

	OnTaskAdded     func(ctx context.Context, meta Meta)
	OnTaskSucceeded func(ctx context.Context, meta Meta)
	OnTaskFailed    func(ctx context.Context, meta Meta, err error)
}

// Task is the function to be executed in a goroutine
type Task func(ctx context.Context) error

// NewExecutor creates a new instance of Background with the provided generic type for the metadata argument.
func NewExecutor[Meta any]() *Executor[Meta] {
	return &Executor[Meta]{}
}

// Run schedules the provided task to be executed in a goroutine. `Meta` is whatever you wish to associate with the
// task.
func (b *Executor[Meta]) Run(ctx context.Context, meta Meta, task Task) {
	b.callOnTaskAdded(ctx, meta)
	b.wg.Add(1)
	go b.run(context.WithoutCancel(ctx), meta, task)
}

// Drain waits for all scheduled tasks to finish
func (b *Executor[Meta]) Drain() {
	b.wg.Wait()
}

func (b *Executor[Meta]) run(ctx context.Context, meta Meta, task Task) {
	defer b.wg.Done()
	err := task(ctx)

	if err != nil {
		b.callOnTaskFailed(ctx, meta, err)
	} else {
		b.callOnTaskSucceeded(ctx, meta)
	}
}

func (b *Executor[Meta]) callOnTaskFailed(ctx context.Context, meta Meta, err error) {
	if b.OnTaskFailed != nil {
		b.OnTaskFailed(ctx, meta, err)
	}
}

func (b *Executor[Meta]) callOnTaskSucceeded(ctx context.Context, meta Meta) {
	if b.OnTaskSucceeded != nil {
		b.OnTaskSucceeded(ctx, meta)
	}
}

func (b *Executor[Meta]) callOnTaskAdded(ctx context.Context, meta Meta) {
	if b.OnTaskAdded != nil {
		b.OnTaskAdded(ctx, meta)
	}
}
