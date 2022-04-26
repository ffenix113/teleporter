package arman92

import (
	"runtime"
	"strconv"
)

type Common struct {
	Client   *Client
	taskType string
	progress int
	details  string
}

func NewCommon(cl *Client, taskType string, details string) *Common {
	return &Common{
		Client:   cl,
		taskType: taskType,
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
	c.details = getCaller() + ": " + err.Error()
}

func (c *Common) SetDone() {
	c.progress = 100
}

func (c *Common) Type() string {
	return c.taskType
}

func (c *Common) Progress() int {
	return c.progress
}

func (c *Common) Details() string {
	return c.details
}
