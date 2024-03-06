package background_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kamilsk/retry/v5/strategy"
	"github.com/stretchr/testify/assert"
	"go.strv.io/background"
)

func Test_NewManager(t *testing.T) {
	m := background.NewManager()
	assert.NotNil(t, m)
	assert.IsType(t, &background.Manager{}, m)
	assert.EqualValues(t, 0, m.CountOf(background.TaskTypeOneOff))
	assert.EqualValues(t, 0, m.CountOf(background.TaskTypeLoop))
}

func Test_RunTaskExecutesInGoroutine(t *testing.T) {
	m := background.NewManager()
	proceed := make(chan bool, 1)

	m.Run(context.Background(), func(ctx context.Context) error {
		// Let the main thread advance a bit
		<-proceed
		proceed <- true
		return nil
	})

	// If the func is not executed in a goroutine the main thread will not be able to advance and the test will time out
	assert.Empty(t, proceed)
	proceed <- true
	m.Wait()
	assert.True(t, <-proceed)
}

func Test_WaitWaitsForPendingTasks(t *testing.T) {
	m := background.NewManager()
	proceed := make(chan bool, 1)
	done := make(chan bool, 1)
	var waited bool

	m.Run(context.Background(), func(ctx context.Context) error {
		// Let the main thread advance a bit
		<-proceed
		return nil
	})

	go func() {
		m.Wait()
		waited = true
		done <- true
	}()

	assert.False(t, waited)
	proceed <- true
	<-done
	assert.True(t, waited)
}

func Test_RunTaskCancelledParentContext(t *testing.T) {
	m := background.NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	proceed := make(chan bool, 1)

	m.Run(ctx, func(ctx context.Context) error {
		<-proceed
		assert.Nil(t, ctx.Err())
		return nil
	})

	cancel()
	proceed <- true
	m.Wait()
}

func Test_OnTaskAdded(t *testing.T) {
	metadata := background.Metadata{"test": "value"}
	executed := false
	var wg sync.WaitGroup
	m := background.NewManagerWithOptions(background.Options{
		Observer: background.DefaultObserver{
			HandleOnTaskAdded: func(ctx context.Context, task background.Task) {
				assert.Equal(t, metadata, task.Meta)
				executed = true
				wg.Done()
			},
		},
	})

	wg.Add(1)
	def := background.Task{
		Fn: func(ctx context.Context) error {
			return nil
		},
		Meta: metadata,
	}
	m.RunTask(context.Background(), def)

	wg.Wait()
	assert.True(t, executed)
}

func Test_OnTaskSucceeded(t *testing.T) {
	metadata := background.Metadata{"test": "value"}
	executed := false
	var wg sync.WaitGroup
	m := background.NewManagerWithOptions(background.Options{
		Observer: background.DefaultObserver{
			HandleOnTaskSucceeded: func(ctx context.Context, task background.Task) {
				assert.Equal(t, metadata, task.Meta)
				executed = true
				wg.Done()
			},
		},
	})

	wg.Add(1)
	def := background.Task{
		Fn: func(ctx context.Context) error {
			return nil
		},
		Meta: metadata,
	}
	m.RunTask(context.Background(), def)

	wg.Wait()
	assert.True(t, executed)
}

func Test_OnTaskFailed(t *testing.T) {
	metadata := background.Metadata{"test": "value"}
	executed := false
	var wg sync.WaitGroup
	m := background.NewManagerWithOptions(background.Options{
		Observer: background.DefaultObserver{
			HandleOnTaskFailed: func(ctx context.Context, task background.Task, err error) {
				assert.Equal(t, metadata, task.Meta)
				assert.Error(t, err)
				executed = true
				wg.Done()
			},
		},
	})

	wg.Add(1)
	def := background.Task{
		Fn: func(ctx context.Context) error {
			return assert.AnError
		},
		Meta: metadata,
	}
	m.RunTask(context.Background(), def)

	wg.Wait()
	assert.True(t, executed)
}

func Test_OnTaskStalled(t *testing.T) {
	tests := []struct {
		duration      time.Duration
		shouldExecute bool
	}{
		{1 * time.Millisecond, false},
		{3 * time.Millisecond, false},
		{6 * time.Millisecond, true},
		{7 * time.Millisecond, true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("duration of %s)", test.duration.String()), func(t *testing.T) {
			metadata := background.Metadata{"test": "value"}
			executed := false
			var wg sync.WaitGroup
			if test.shouldExecute == true {
				wg.Add(1)
			}

			m := background.NewManagerWithOptions(background.Options{
				StalledThreshold: time.Millisecond * 5,
				Observer: background.DefaultObserver{
					HandleOnTaskStalled: func(ctx context.Context, task background.Task) {
						assert.Equal(t, metadata, task.Meta)
						executed = true
						wg.Done()
					},
				},
			})

			def := background.Task{
				Fn: func(ctx context.Context) error {
					<-time.After(test.duration)
					return nil
				},
				Meta: metadata,
			}
			m.RunTask(context.Background(), def)
			m.Run(context.Background(), func(ctx context.Context) error {
				return nil
			})

			wg.Wait()
			assert.Equal(t, test.shouldExecute, executed)
		})
	}
}

func Test_StalledTaskStillCallsOnTaskSucceeded(t *testing.T) {
	executed := false
	var wg sync.WaitGroup
	m := background.NewManagerWithOptions(background.Options{
		StalledThreshold: time.Millisecond,
		Observer: background.DefaultObserver{
			HandleOnTaskSucceeded: func(ctx context.Context, task background.Task) {
				executed = true
				wg.Done()
			},
		},
	})

	wg.Add(1)
	m.Run(context.Background(), func(ctx context.Context) error {
		<-time.After(time.Millisecond * 3)
		return nil
	})

	wg.Wait()
	assert.True(t, executed)
}

func Test_TaskRetryStrategies(t *testing.T) {
	var limit uint = 5
	var count uint = 0
	m := background.NewManager()
	def := background.Task{
		Fn: func(ctx context.Context) error {
			count++
			return assert.AnError
		},
		Retry: background.Retry{
			strategy.Limit(limit),
		},
	}

	m.RunTask(context.Background(), def)
	m.Wait()

	assert.Equal(t, limit, count)
}

func Test_ManagerRetryStrategies(t *testing.T) {
	var limit uint = 5
	var count uint = 0
	m := background.NewManagerWithOptions(background.Options{
		Retry: background.Retry{
			strategy.Limit(limit),
		},
	})

	m.Run(context.Background(), func(ctx context.Context) error {
		count++
		return assert.AnError
	})
	m.Wait()

	assert.Equal(t, limit, count)
}

func Test_RunTaskTypeLoop(t *testing.T) {
	loops := 0
	m := background.NewManager()
	def := background.Task{
		Type: background.TaskTypeLoop,
		Fn: func(ctx context.Context) error {
			loops++
			return nil
		},
	}

	m.RunTask(context.Background(), def)
	<-time.After(time.Microsecond * 500)

	m.Cancel()
	assert.GreaterOrEqual(t, loops, 100)
}

func Test_RunTaskTypeLoop_RetryStrategies(t *testing.T) {
	done := make(chan error, 1)
	count := 0

	m := background.NewManagerWithOptions(background.Options{
		Observer: background.DefaultObserver{
			HandleOnTaskFailed: func(ctx context.Context, task background.Task, err error) {
				done <- err
			},
		},
	})
	def := background.Task{
		Type: background.TaskTypeLoop,
		Fn: func(ctx context.Context) error {
			count++
			// TODO: Figure out why we need to wait here to avoid test timeout
			<-time.After(time.Millisecond)
			return assert.AnError
		},
		Retry: background.Retry{
			strategy.Limit(2),
		},
	}

	m.RunTask(context.Background(), def)
	err := <-done
	m.Cancel()

	assert.Equal(t, assert.AnError, err)
	// We cannot guarantee exact count of executions because by the time we cancel the task the loop might have made
	// several additional iterations.
	assert.GreaterOrEqual(t, count, 2)
}

func Test_RunTaskTypeLoop_CancelledParentContext(t *testing.T) {
	m := background.NewManager()
	cancellable, cancel := context.WithCancel(context.Background())
	proceed := make(chan bool, 1)
	done := make(chan error, 1)
	var once sync.Once

	def := background.Task{
		Type: background.TaskTypeLoop,
		Fn: func(ctx context.Context) error {
			once.Do(func() {
				proceed <- true
				// Cancel the parent context and send the child context's error out to the test
				// The expectation is that the child context will not be cancelled
				cancel()
				done <- ctx.Err()
			})

			return nil
		},
	}

	m.RunTask(cancellable, def)
	// Make sure we wait for the loop to run at least one iteration before cancelling it
	<-proceed
	m.Cancel()
	err := <-done

	assert.Equal(t, nil, err)
}

func Test_CountOf(t *testing.T) {
	m := background.NewManager()

	assert.Equal(t, 0, m.CountOf(background.TaskTypeOneOff))
	assert.Equal(t, 0, m.CountOf(background.TaskTypeLoop))
	assert.Equal(t, 0, m.CountOf(background.TaskType(3)))

	def := background.Task{
		Type: background.TaskTypeOneOff,
		Fn: func(ctx context.Context) error {
			return nil
		},
	}
	m.RunTask(context.Background(), def)
	assert.Equal(t, 1, m.CountOf(background.TaskTypeOneOff))
	assert.Equal(t, 0, m.CountOf(background.TaskTypeLoop))
	assert.Equal(t, 0, m.CountOf(background.TaskType(3)))
	m.Wait()

	def = background.Task{
		Type: background.TaskTypeLoop,
		Fn: func(ctx context.Context) error {
			return nil
		},
	}

	m.RunTask(context.Background(), def)
	assert.Equal(t, 0, m.CountOf(background.TaskTypeOneOff))
	assert.Equal(t, 1, m.CountOf(background.TaskTypeLoop))
	assert.Equal(t, 0, m.CountOf(background.TaskType(3)))
	m.Cancel()

	assert.Equal(t, 0, m.CountOf(background.TaskTypeOneOff))
	assert.Equal(t, 0, m.CountOf(background.TaskTypeLoop))
	assert.Equal(t, 0, m.CountOf(background.TaskType(3)))
}
