package background_test

import (
	"context"
	"testing"

	"github.com/robertrossmann/background"
	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	b := background.NewExecutor[bool]()
	assert.NotNil(t, b)
	assert.IsType(t, &background.Executor[bool]{}, b)
	assert.Nil(t, b.OnTaskAdded)
	assert.Nil(t, b.OnTaskSucceeded)
	assert.Nil(t, b.OnTaskFailed)
}

func Test_RunExecutesInGoroutine(t *testing.T) {
	b := background.NewExecutor[bool]()
	proceed := make(chan bool, 1)

	b.Run(context.Background(), true, func(ctx context.Context) error {
		// Let the main thread advance a bit
		<-proceed
		proceed <- true
		return nil
	})

	// If the func is not executed in a goroutine the main thread will not be able to advance and the test will time out
	assert.Empty(t, proceed)
	proceed <- true
	b.Drain()
	assert.True(t, <-proceed)
}

func Test_DrainWaitsForPendingTasks(t *testing.T) {
	b := background.NewExecutor[bool]()
	proceed := make(chan bool, 1)
	done := make(chan bool, 1)
	var drained bool

	b.Run(context.Background(), true, func(ctx context.Context) error {
		// Let the main thread advance a bit
		<-proceed
		return nil
	})

	go func() {
		b.Drain()
		drained = true
		done <- true
	}()

	assert.False(t, drained)
	proceed <- true
	<-done
	assert.True(t, drained)
}

func Test_CancelledParentContext(t *testing.T) {
	b := background.NewExecutor[bool]()
	ctx, cancel := context.WithCancel(context.Background())
	proceed := make(chan bool, 1)

	b.Run(ctx, true, func(ctx context.Context) error {
		<-proceed
		assert.Nil(t, ctx.Err())
		return nil
	})

	cancel()
	proceed <- true
	b.Drain()
}

func Test_Len(t *testing.T) {
	b := background.NewExecutor[bool]()
	proceed := make(chan bool, 1)
	remaining := 10

	b.OnTaskSucceeded = func(ctx context.Context, meta bool) {
		assert.Equal(t, remaining, b.Len())
		remaining--
		proceed <- true
	}

	for range 10 {
		b.Run(context.Background(), true, func(ctx context.Context) error {
			<-proceed
			return nil
		})
	}

	proceed <- true
	b.Drain()
	assert.Equal(t, 0, b.Len())
}

func Test_OnTaskAdded(t *testing.T) {
	b := background.NewExecutor[bool]()
	metaval := true
	executed := false

	b.OnTaskAdded = func(ctx context.Context, meta bool) {
		assert.Equal(t, metaval, meta)
		executed = true
	}

	b.Run(context.Background(), metaval, func(ctx context.Context) error {
		return nil
	})
	b.Drain()
	assert.True(t, executed)
}

func Test_OnTaskSucceeded(t *testing.T) {
	b := background.NewExecutor[bool]()
	metaval := true
	executed := false

	b.OnTaskSucceeded = func(ctx context.Context, meta bool) {
		assert.Equal(t, metaval, meta)
		executed = true
	}

	b.Run(context.Background(), metaval, func(ctx context.Context) error {
		return nil
	})
	b.Drain()
	assert.True(t, executed)
}

func Test_OnTaskFailed(t *testing.T) {
	b := background.NewExecutor[bool]()
	metaval := true
	executed := false

	b.OnTaskFailed = func(ctx context.Context, meta bool, err error) {
		assert.Equal(t, metaval, meta)
		assert.Error(t, err)
		executed = true
	}

	b.Run(context.Background(), metaval, func(ctx context.Context) error {
		return assert.AnError
	})
	b.Drain()
	assert.True(t, executed)
}
