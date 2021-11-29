package tasks

import (
	"context"
	"fmt"
	"time"
)

const (
	TaskStatusNew TaskStatus = iota
	TaskStatusInProgress
	TaskStatusDone
	TaskStatusError
)

type TaskStatus int

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusNew:
		return "new"
	case TaskStatusInProgress:
		return "in progress"
	case TaskStatusDone:
		return "done"
	case TaskStatusError:
		return "error"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

type Task interface {
	Type() string
	Name() string
	Run(ctx context.Context)
	Progress() int
	Status() TaskStatus
	Details() string
}

type Monitor struct {
	Input chan Task
	tasks []Task
}

func NewMonitor(ctx context.Context, taskChanCap int) *Monitor {
	m := &Monitor{
		Input: make(chan Task, taskChanCap),
	}

	go m.Run(ctx)

	return m
}

func (m *Monitor) Run(ctx context.Context) {
	for ctx.Err() == nil {
		select {
		case task := <-m.Input:
			m.tasks = append(m.tasks, task)

			task.Run(ctx)
			// TODO: maybe restart if task failed
			time.Sleep(time.Second)
		case <-ctx.Done():
			return
		}
	}
}

func (m *Monitor) List(offset, limit int) []Task {
	tasks := m.tasks

	if offset >= len(tasks) {
		return nil
	}

	start := tasks[offset:]
	if limit > len(start) {
		limit = len(start)
	}

	return start[:limit]
}
