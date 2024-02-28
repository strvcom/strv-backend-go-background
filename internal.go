package background

import (
	"time"

	"github.com/kamilsk/retry/v5/strategy"
)

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
