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

type testmeta bool

func Test_New(t *testing.T) {
	m := background.NewManager()
	assert.NotNil(t, m)
	assert.IsType(t, &background.Manager[background.NilMeta]{}, m)
	assert.Equal(t, 0, m.Len())
}

func Test_RunExecutesInGoroutine(t *testing.T) {
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

func Test_CancelledParentContext(t *testing.T) {
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

func Test_Len(t *testing.T) {
	proceed := make(chan bool, 1)
	remaining := 10
	m := background.NewManagerWithOptions(background.Options[background.NilMeta]{
		Hooks: background.Hooks[background.NilMeta]{
			OnTaskSucceeded: func(ctx context.Context, meta background.NilMeta) {
				remaining--
				proceed <- true
			},
		},
	})

	for range 10 {
		m.Run(context.Background(), func(ctx context.Context) error {
			<-proceed
			return nil
		})
	}

	proceed <- true
	m.Wait()
	assert.Equal(t, 0, m.Len())
}

func Test_OnTaskAdded(t *testing.T) {
	var metaval testmeta = true
	executed := false
	var wg sync.WaitGroup
	m := background.NewManagerWithOptions(background.Options[testmeta]{
		Hooks: background.Hooks[testmeta]{
			OnTaskAdded: func(ctx context.Context, meta testmeta) {
				assert.Equal(t, metaval, meta)
				executed = true
				wg.Done()
			},
		},
	})

	wg.Add(1)
	def := background.TaskDefinition[testmeta]{
		Task: func(ctx context.Context) error {
			return nil
		},
		Meta: metaval,
	}
	m.RunTaskDefinition(context.Background(), def)

	wg.Wait()
	assert.True(t, executed)
}

func Test_OnTaskSucceeded(t *testing.T) {
	var metaval testmeta = true
	executed := false
	var wg sync.WaitGroup
	m := background.NewManagerWithOptions(background.Options[testmeta]{
		Hooks: background.Hooks[testmeta]{
			OnTaskSucceeded: func(ctx context.Context, meta testmeta) {
				assert.Equal(t, metaval, meta)
				executed = true
				wg.Done()
			},
		},
	})

	wg.Add(1)
	def := background.TaskDefinition[testmeta]{
		Task: func(ctx context.Context) error {
			return nil
		},
		Meta: metaval,
	}
	m.RunTaskDefinition(context.Background(), def)

	wg.Wait()
	assert.True(t, executed)
}

func Test_OnTaskFailed(t *testing.T) {
	var metaval testmeta = true
	executed := false
	var wg sync.WaitGroup
	m := background.NewManagerWithOptions(background.Options[testmeta]{
		Hooks: background.Hooks[testmeta]{
			OnTaskFailed: func(ctx context.Context, meta testmeta, err error) {
				assert.Equal(t, metaval, meta)
				assert.Error(t, err)
				executed = true
				wg.Done()
			},
		},
	})

	wg.Add(1)
	def := background.TaskDefinition[testmeta]{
		Task: func(ctx context.Context) error {
			return assert.AnError
		},
		Meta: metaval,
	}
	m.RunTaskDefinition(context.Background(), def)

	wg.Wait()
	assert.True(t, executed)
}

func Test_OnGoroutineStalled(t *testing.T) {
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
			var metaval testmeta = true
			executed := false
			var wg sync.WaitGroup
			if test.shouldExecute == true {
				wg.Add(1)
			}

			m := background.NewManagerWithOptions(background.Options[testmeta]{
				StalledThreshold: time.Millisecond * 5,
				Hooks: background.Hooks[testmeta]{
					OnGoroutineStalled: func(ctx context.Context, meta testmeta) {
						assert.Equal(t, metaval, meta)
						executed = true
						wg.Done()
					},
				},
			})

			def := background.TaskDefinition[testmeta]{
				Task: func(ctx context.Context) error {
					<-time.After(test.duration)
					return nil
				},
				Meta: metaval,
			}
			m.RunTaskDefinition(context.Background(), def)
			m.Run(context.Background(), func(ctx context.Context) error {
				return nil
			})

			wg.Wait()
			assert.Equal(t, test.shouldExecute, executed)
		})
	}
}

func Test_StalledGoroutineStillCallsOnTaskSucceeded(t *testing.T) {
	executed := false
	var wg sync.WaitGroup
	m := background.NewManagerWithOptions(background.Options[testmeta]{
		StalledThreshold: time.Millisecond,
		Hooks: background.Hooks[testmeta]{
			OnTaskSucceeded: func(ctx context.Context, meta testmeta) {
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

func Test_TaskDefinitionRetryStrategies(t *testing.T) {
	var limit uint = 5
	var count uint = 0
	m := background.NewManager()
	def := background.TaskDefinition[background.NilMeta]{
		Task: func(ctx context.Context) error {
			count++
			return assert.AnError
		},
		Retry: background.Retry{
			strategy.Limit(limit),
		},
	}

	m.RunTaskDefinition(context.Background(), def)
	m.Wait()

	assert.Equal(t, limit, count)
}

func Test_ManagerDefaultRetryStrategies(t *testing.T) {
	var limit uint = 5
	var count uint = 0
	m := background.NewManagerWithOptions(background.Options[background.NilMeta]{
		DefaultRetry: background.Retry{
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
