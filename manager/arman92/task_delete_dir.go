package arman92

import (
	"context"
	"strings"

	"github.com/ffenix113/teleporter/tasks"
)

type DeleteDir struct {
	*Common
	RelativeDirPath string
}

func NewDeleteDir(cl *Client, dirPath string) *DeleteDir {
	return &DeleteDir{
		Common: &Common{
			Client:   cl,
			taskType: "DeleteDir",
		},
		// Add slash to signify that it is a directory.
		RelativeDirPath: cl.RelativePath(dirPath) + "/",
	}
}

func (d *DeleteDir) Name() string {
	return d.RelativeDirPath
}

func (d *DeleteDir) Run(ctx context.Context) {
	d.status = tasks.TaskStatusInProgress

	var msgsToDelete []int64
	for filePath, msgID := range d.Client.PinnedHeader.Files {
		if !strings.HasPrefix(filePath, d.RelativeDirPath) {
			continue
		}

		delete(d.Client.PinnedHeader.Files, filePath)
		msgsToDelete = append(msgsToDelete, msgID)
	}
	// Can also be done as a separate tasks to delete provided files
	// by using DeleteFile.
	_, err := d.Client.TDClient.DeleteMessages(d.Client.chatID, msgsToDelete, true)
	if err != nil {
		d.SetError(err)
		return
	}

	d.Client.FileTree.Delete(d.RelativeDirPath)
	if err := d.Client.SendHeader(ctx); err != nil {
		d.SetError(err)
		return
	}

	d.SetDone()
}
