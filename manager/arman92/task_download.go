package arman92

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/Arman92/go-tdlib/v2/tdlib"

	"github.com/ffenix113/teleporter/tasks"
)

type DownloadFile struct {
	*Common
	RelativePath string
}

// NewDownloadFile will return a download file task.
//
// filePath can be absolute or relative.
func NewDownloadFile(cl *Client, filePath string, details ...string) *DownloadFile {
	return &DownloadFile{
		Common: &Common{
			Client:   cl,
			taskType: "DownloadFile",
			details:  detailsOrEmpty(details...),
		},
		RelativePath: cl.RelativePath(filePath),
	}
}

func (f *DownloadFile) Name() string {
	return f.RelativePath
}

func (f *DownloadFile) Run(ctx context.Context) {
	f.status = tasks.TaskStatusInProgress

	msgID, ok := f.Client.PinnedHeader.Files[f.RelativePath]
	if !ok {
		f.SetError(fmt.Errorf("file %q is not present in remote chat", f.RelativePath))
		return
	}

	if err := f.Client.EnsureMessagesAreKnown(ctx, msgID); err != nil {
		f.SetError(err)
		return
	}

	msg, err := f.Client.TDClient.GetMessage(f.Client.chatID, msgID)
	if err != nil {
		f.SetError(err)
		return
	}

	msgDoc, ok := msg.Content.(*tdlib.MessageDocument)
	if !ok {
		f.SetError(fmt.Errorf("message is not document: %v", msg.Content))
		return
	}

	fileID := msgDoc.Document.Document.ID

	var filePath string
	if file, err := f.Client.TDClient.GetFile(fileID); err == nil {
		if !file.Local.IsDownloadingCompleted {
			if _, err := f.Client.TDClient.CancelDownloadFile(fileID, false); err != nil {
				f.SetError(err)
				return
			}

			watcher := f.watchDownload(fileID) // This may dangle if download will screw up.
			// Download(msgDoc.Document.Document.ID, DownloadFilePartSize)
			_, err = f.Client.TDClient.DownloadFile(fileID, 1, 0, 0, false)
			if err != nil {
				f.SetError(err)
				return
			}

			file = <-watcher
		}
		filePath = file.Local.Path
	}

	if err := os.MkdirAll(path.Dir(f.Client.AbsPath(f.RelativePath)), os.ModeDir|0755); err != nil {
		f.SetError(err)
		return
	}

	if err := os.Rename(filePath, f.Client.AbsPath(f.RelativePath)); err != nil {
		f.SetError(fmt.Errorf("move file: %w", err))
	}

	f.SetDone()
}

func (f *DownloadFile) Download(fileID int32, partSize int32) (*tdlib.File, error) {
	var offset int32
	var file *tdlib.File
	var err error

	for {
		file, err = f.Client.TDClient.DownloadFile(fileID, 1, offset, partSize, true)
		if err != nil {
			return nil, err
		}

		if file.Local.IsDownloadingCompleted {
			return file, nil
		}

		f.progress = int(100 * (file.Local.DownloadedSize / file.ExpectedSize))

		offset += partSize
	}
}

func (f *DownloadFile) watchDownload(fileID int32) chan *tdlib.File {
	watcher := make(chan *tdlib.File, 1)
	var fileUpdate tdlib.UpdateFile

	f.Client.AddUpdateHandler(func(update tdlib.UpdateMsg) bool {
		if update.Data["@type"] != string(tdlib.UpdateFileType) {
			return false
		}

		if err := json.Unmarshal(update.Raw, &fileUpdate); err != nil {
			panic(fmt.Sprintf("failed to unmarshal update: %v", err))
		}
		if fileUpdate.File.ID != fileID {
			return false
		}

		f.progress = int(100 * (float64(fileUpdate.File.Local.DownloadedSize) / float64(fileUpdate.File.ExpectedSize)))

		if fileUpdate.File.Local.IsDownloadingCompleted {
			watcher <- fileUpdate.File
			close(watcher)
			return true
		}

		return false
	})

	return watcher
}
