package arman92

import "context"

type StaticTask struct {
	*Common
	RelativePath string
}

func NewStaticTask(filePath string, commonData *Common) StaticTask {
	return StaticTask{
		Common:       commonData,
		RelativePath: filePath,
	}
}

func (t StaticTask) Name() string {
	return t.RelativePath
}

func (StaticTask) Run(_ context.Context) {}
