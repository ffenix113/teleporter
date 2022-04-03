package arman92

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/ffenix113/teleporter/tasks"
)

type DeleteFile struct {
	*Common
	RelativePath string
}

func NewDeleteFile(cl *Client, filePath string) *DeleteFile {
	return &DeleteFile{
		Common: &Common{
			Client:   cl,
			taskType: "DeleteFile",
		},
		RelativePath: cl.RelativePath(filePath),
	}
}

func (f *DeleteFile) Name() string {
	return f.RelativePath
}

func (f *DeleteFile) Run(ctx context.Context) {
	f.status = tasks.TaskStatusInProgress

	msgID, ok := f.Client.PinnedHeader.Files[f.RelativePath]
	if !ok {
		f.SetError(fmt.Errorf("file is not present in header: %q", f.RelativePath))
		return
	}

	// Remove physical file fist so if this fails - we will be able to re-fetch file later.
	absFilePath := f.Client.AbsPath(f.RelativePath)
	_, err := os.Stat(absFilePath)
	if err == nil || !errors.Is(err, fs.ErrNotExist) {
		if err := os.Remove(absFilePath); err != nil {
			f.SetError(err)
			return
		}
	}

	// It is safer to have deleted file and entry left in header
	// than the other way around. If header entry will be missing for a file
	// files will be leaking.
	_, err = f.Client.TDClient.DeleteMessages(f.Client.chatID, []int64{msgID}, true)
	if err != nil {
		f.SetError(err)
		return
	}

	delete(f.Client.PinnedHeader.Files, f.RelativePath)
	f.Client.FileTree.Delete(f.RelativePath)
	if err := f.Client.SendHeader(ctx); err != nil {
		f.SetError(err)
		return
	}

	f.SetDone()
}
