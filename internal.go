package background

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kamilsk/retry/v5/strategy"
)

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
func mkstrategies(defaults Retry, overrides Retry) Retry {
	result := make(Retry, 0, max(len(defaults), len(overrides), 1))

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
