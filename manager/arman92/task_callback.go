package arman92

import (
	"context"
)

type Task interface{ Run(ctx context.Context) }

type Callback struct {
	Task
	done func(task Task)
}

// WithCallback will execute callback when task is done.
//
// Note: if task will be restarted - callback will be executed again.
func WithCallback(task Task, callback func(task Task)) Callback {
	return Callback{
		Task: task,
		done: callback,
	}
}

func (c Callback) Run(ctx context.Context) {
	c.Task.Run(ctx)
	c.done(c.Task)
}
