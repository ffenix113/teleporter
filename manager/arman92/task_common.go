package arman92

import (
	"runtime"
	"strconv"

	"github.com/ffenix113/teleporter/tasks"
)

type Common struct {
	Client   *Client
	taskType string
	status   tasks.TaskStatus
	progress int
	details  string
}

func NewCommon(cl *Client, taskType string, status tasks.TaskStatus, details string) *Common {
	return &Common{
		Client:   cl,
		taskType: taskType,
		status:   status,
		details:  details,
		progress: 100,
	}
}

func getCaller() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "<noLine>"
	}

	return file + ":" + strconv.Itoa(line)
}

func (c *Common) SetError(err error) {
	c.progress = 100
	c.status = tasks.TaskStatusError
	c.details = getCaller() + ": " + err.Error()
}

func (c *Common) SetDone() {
	c.progress = 100
	c.status = tasks.TaskStatusDone
}

func (c *Common) Type() string {
	return c.taskType
}

func (c *Common) Progress() int {
	return c.progress
}

func (c *Common) Status() tasks.TaskStatus {
	return c.status
}

func (c *Common) Details() string {
	return c.details
}

func detailsOrEmpty(strs ...string) string {
	if len(strs) == 0 {
		return ""
	}

	return strs[0]
}
