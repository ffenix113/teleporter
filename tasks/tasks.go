package tasks

import (
	"context"
	"fmt"
	"sync"
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

type Hook func(task Task) (Task, bool, error)

type Monitor struct {
	preAddHook []Hook

	tasks   []Task
	tasksMu sync.Mutex
}

func NewMonitor(ctx context.Context) *Monitor {
	m := &Monitor{}

	go m.Run(ctx)

	return m
}

func (m *Monitor) AddTask(task Task) {
	m.tasksMu.Lock()
	var err error
	var finished bool
	for i, h := range m.preAddHook {
		task, finished, err = h(task)
		if err != nil {
			panic(fmt.Errorf("pre-add hook: %w", err))
		}

		if finished {
			// FIXME: panics on 'i' here
			m.preAddHook = append(m.preAddHook[:i], m.preAddHook[i+1:]...)
		}
	}

	m.tasks = append(m.tasks, task)
	m.tasksMu.Unlock()
}

func (m *Monitor) AddPreAddHook(hook Hook) {
	m.tasksMu.Lock()
	defer m.tasksMu.Unlock()

	m.preAddHook = append(m.preAddHook, hook)
}

func (m *Monitor) Run(ctx context.Context) {
	var taskIdx int
	var task Task

	for ctx.Err() == nil {
		m.tasksMu.Lock()
		if taskIdx == len(m.tasks) {
			m.tasksMu.Unlock()
			time.Sleep(500 * time.Millisecond)
			continue
		}

		task = m.tasks[taskIdx]
		m.tasksMu.Unlock()

		task.Run(ctx)

		taskIdx++
		// TODO: maybe restart if task failed
		time.Sleep(2 * time.Second)
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
