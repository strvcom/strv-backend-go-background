package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	type args struct {
		ctx  context.Context
		opts []Option
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "pass",
			args: args{
				ctx: ctx,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.args.ctx, tt.args.opts...)
			assert.NotNil(t, m)
			assert.Equal(t, 0, m.stat.RunningTasks)
			assert.Nil(t, m.retrier)
		})
	}
}

func TestNewWithRetry(t *testing.T) {
	ctx := context.Background()
	type args struct {
		ctx  context.Context
		opts []Option
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "pass:with-constant-retry",
			args: args{
				ctx: ctx,
				opts: []Option{
					WithConstantRetry(5, 1*time.Second),
				},
			},
		},
		{
			name: "pass:with-expotential-retry",
			args: args{
				ctx: ctx,
				opts: []Option{
					WithExpotentialRetry(5, 250*time.Second, 1*time.Second),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.args.ctx, tt.args.opts...)
			assert.NotNil(t, m)
			assert.NotNil(t, m.retrier)
		})
	}
}

func TestRunWithRetry(t *testing.T) {
	expectedAttempts := 5
	ctx := context.Background()
	type args struct {
		ctx  context.Context
		opts []Option
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "pass:with-constant-retry",
			args: args{
				ctx: ctx,
				opts: []Option{
					WithConstantRetry(5, 1*time.Second),
				},
			},
		},
		{
			name: "pass:with-expotential-retry",
			args: args{
				ctx: ctx,
				opts: []Option{
					WithExpotentialRetry(5, 250*time.Second, 1*time.Second),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.args.ctx, tt.args.opts...)
			assert.NotNil(t, m)
			assert.NotNil(t, m.retrier)
			nRuns := -1
			m.Run(func(ctx context.Context) error {
				nRuns++
				<-time.After(1 * time.Second)
				return errors.New("test")
			})
			assert.Equal(t, 1, m.stat.RunningTasks)
			err := m.Wait()
			assert.Error(t, err)
			assert.ErrorContains(t, err, "test")
			assert.Equal(t, expectedAttempts, nRuns)
		})
	}
}

func TestContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type args struct {
		ctx  context.Context
		opts []Option
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "pass",
			args: args{
				ctx: ctx,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.args.ctx, tt.args.opts...)
			assert.NotNil(t, m)
			m.Run(func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Second):
					return errors.New("timeout")
				}
			})
			assert.Equal(t, 1, m.stat.RunningTasks)
			go time.AfterFunc(1*time.Second, func() {
				cancel()
			})
			err := m.Wait()
			assert.NotNil(t, err)
			assert.ErrorContains(t, err, "context canceled")
			assert.Equal(t, 0, m.stat.RunningTasks)
		})
	}
}

func TestStop(t *testing.T) {
	type args struct {
		ctx  context.Context
		opts []Option
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "pass",
			args: args{
				ctx: context.Background(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.args.ctx, tt.args.opts...)
			assert.NotNil(t, m)
			m.Run(func(ctx context.Context) error {
				select {
				case <-ctx.Done():
				case <-time.After(1 * time.Second):
					return errors.New("timeout")
				}
				return nil
			})
			assert.Equal(t, 1, m.stat.RunningTasks)
			err := m.Stop()
			assert.Nil(t, err)
			assert.Equal(t, 0, m.stat.RunningTasks)
		})
	}
}
