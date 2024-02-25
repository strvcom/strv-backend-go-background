package background_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.strv.io/background"
)

func Test_New(t *testing.T) {
	m := background.NewManager[bool]()
	assert.NotNil(t, m)
	assert.IsType(t, &background.Manager[bool]{}, m)
	assert.Equal(t, m.StalledThreshold, time.Duration(0))
	assert.Nil(t, m.OnTaskAdded)
	assert.Nil(t, m.OnTaskSucceeded)
	assert.Nil(t, m.OnTaskFailed)
	assert.Nil(t, m.OnGoroutineStalled)
}

func Test_RunExecutesInGoroutine(t *testing.T) {
	m := background.NewManager[bool]()
	proceed := make(chan bool, 1)

	m.Run(context.Background(), true, func(ctx context.Context) error {
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
	m := background.NewManager[bool]()
	proceed := make(chan bool, 1)
	done := make(chan bool, 1)
	var waited bool

	m.Run(context.Background(), true, func(ctx context.Context) error {
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
	m := background.NewManager[bool]()
	ctx, cancel := context.WithCancel(context.Background())
	proceed := make(chan bool, 1)

	m.Run(ctx, true, func(ctx context.Context) error {
		<-proceed
		assert.Nil(t, ctx.Err())
		return nil
	})

	cancel()
	proceed <- true
	m.Wait()
}

func Test_Len(t *testing.T) {
	m := background.NewManager[bool]()
	proceed := make(chan bool, 1)
	remaining := 10

	m.OnTaskSucceeded = func(ctx context.Context, meta bool) {
		remaining--
		assert.Equal(t, remaining, m.Len())
		proceed <- true
	}

	for range 10 {
		m.Run(context.Background(), true, func(ctx context.Context) error {
			<-proceed
			return nil
		})
	}

	proceed <- true
	m.Wait()
	assert.Equal(t, 0, m.Len())
}

func Test_OnTaskAdded(t *testing.T) {
	m := background.NewManager[bool]()
	metaval := true
	executed := false

	m.OnTaskAdded = func(ctx context.Context, meta bool) {
		assert.Equal(t, metaval, meta)
		executed = true
	}

	m.Run(context.Background(), metaval, func(ctx context.Context) error {
		return nil
	})
	m.Wait()
	assert.True(t, executed)
}

func Test_OnTaskSucceeded(t *testing.T) {
	m := background.NewManager[bool]()
	metaval := true
	executed := false

	m.OnTaskSucceeded = func(ctx context.Context, meta bool) {
		assert.Equal(t, metaval, meta)
		executed = true
	}

	m.Run(context.Background(), metaval, func(ctx context.Context) error {
		return nil
	})
	m.Wait()
	assert.True(t, executed)
}

func Test_OnTaskFailed(t *testing.T) {
	m := background.NewManager[bool]()
	metaval := true
	executed := false

	m.OnTaskFailed = func(ctx context.Context, meta bool, err error) {
		assert.Equal(t, metaval, meta)
		assert.Error(t, err)
		executed = true
	}

	m.Run(context.Background(), metaval, func(ctx context.Context) error {
		return assert.AnError
	})
	m.Wait()
	assert.True(t, executed)
}

func Test_OnGoroutineStalled(t *testing.T) {
	tests := []struct {
		duration      time.Duration
		shouldExecute bool
	}{
		{time.Duration(1 * time.Millisecond), false},
		{time.Duration(3 * time.Millisecond), false},
		{time.Duration(6 * time.Millisecond), true},
		{time.Duration(7 * time.Millisecond), true},
	}

	for _, test := range tests {
		m := background.NewManager[bool]()
		m.StalledThreshold = time.Millisecond * 5

		t.Run(fmt.Sprintf("duration of %s)", test.duration.String()), func(t *testing.T) {
			var wg sync.WaitGroup
			executed := false
			if test.shouldExecute == true {
				wg.Add(1)
			}

			m.OnGoroutineStalled = func(ctx context.Context, meta bool) {
				executed = true
				wg.Done()
			}

			m.Run(context.Background(), true, func(ctx context.Context) error {
				<-time.After(test.duration)
				return nil
			})

			wg.Wait()
			m.Wait()
			assert.Equal(t, test.shouldExecute, executed)
		})
	}
}

func Test_StalledGoroutineStillCallsOnTaskSucceeded(t *testing.T) {
	m := background.NewManager[bool]()
	m.StalledThreshold = time.Millisecond
	executed := false

	m.OnTaskSucceeded = func(ctx context.Context, meta bool) {
		executed = true
	}

	m.Run(context.Background(), true, func(ctx context.Context) error {
		<-time.After(time.Millisecond * 3)
		return nil
	})

	m.Wait()
	assert.True(t, executed)
}
