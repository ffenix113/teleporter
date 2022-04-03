package arman92

import (
	"context"

	"github.com/ffenix113/teleporter/tasks"
)

type Callback struct {
	tasks.Task
	done func(task tasks.Task)
}

// WithCallback will execute callback when task is done.
//
// Note: if task will be restarted - callback will be executed again.
func WithCallback(task tasks.Task, callback func(task tasks.Task)) Callback {
	return Callback{
		Task: task,
		done: callback,
	}
}

func (c Callback) Run(ctx context.Context) {
	c.Task.Run(ctx)
	c.done(c.Task)
}
