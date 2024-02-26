package manager

import (
	"time"

	"github.com/eapache/go-resiliency/retrier"
)

type Option interface {
	Apply(*Manager)
}

func WithCancelOnError() Option {
	return &OptionWithCancelOnError{}
}

type OptionWithCancelOnError struct{}

func (o *OptionWithCancelOnError) Apply(m *Manager) {
	*m.pool = *m.pool.WithCancelOnError()
}

func WithFirstError() Option {
	return &OptionWithFirstError{}
}

type OptionWithFirstError struct{}

func (o *OptionWithFirstError) Apply(m *Manager) {
	*m.pool = *m.pool.WithFirstError()
}

func WithMaxTasks(n int) Option {
	return &OptionWithMaxTasks{n}
}

type OptionWithMaxTasks struct {
	n int
}

func (o *OptionWithMaxTasks) Apply(m *Manager) {
	*m.pool = *m.pool.WithMaxGoroutines(o.n)
}

func WithConstantRetry(attempts int, backoffDuration time.Duration) Option {
	return &OptionWithConstantRetry{
		attempts:        attempts,
		backoffDuration: backoffDuration,
	}
}

type OptionWithConstantRetry struct {
	attempts        int
	backoffDuration time.Duration
}

func (o *OptionWithConstantRetry) Apply(m *Manager) {
	backoff := retrier.ConstantBackoff(o.attempts, o.backoffDuration)
	m.retrier = retrier.New(backoff, retrier.DefaultClassifier{})
}

func WithExpotentialRetry(attempts int, minBackoffDuration time.Duration, maxBackoffDuration time.Duration) Option {
	return &OptionWithExpotentialRetry{
		attempts:           attempts,
		minBackoffDuration: minBackoffDuration,
		maxBackoffDuration: maxBackoffDuration,
	}
}

type OptionWithExpotentialRetry struct {
	attempts           int
	minBackoffDuration time.Duration
	maxBackoffDuration time.Duration
}

func (o *OptionWithExpotentialRetry) Apply(m *Manager) {
	backoff := retrier.LimitedExponentialBackoff(o.attempts, o.minBackoffDuration, o.maxBackoffDuration)
	m.retrier = retrier.New(backoff, retrier.DefaultClassifier{})
}
