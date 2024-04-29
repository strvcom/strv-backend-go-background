package task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.strv.io/background/task"
)

func Test_New(t *testing.T) {
	called := false
	def := task.New(task.TypeOneOff, func(ctx context.Context) error {
		called = true
		return nil
	})

	err := def.Fn(context.Background())

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, task.TypeOneOff, def.Type)

	assert.Empty(t, def.Meta)
	assert.Empty(t, def.Retry)
}
