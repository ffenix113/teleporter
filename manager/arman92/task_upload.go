package arman92

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Arman92/go-tdlib/v2/tdlib"

	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/tasks"
)

type UploadFile struct {
	*Common
	RelativePath  string
	FileUpdatedAt time.Time
}

func NewUploadFile(cl *Client, filePath string, description ...string) *UploadFile {
	stat, _ := os.Stat(filePath)

	return &UploadFile{
		Common: &Common{
			Client:   cl,
			taskType: "UploadFile",
			details:  detailsOrEmpty(description...),
		},
		RelativePath:  cl.RelativePath(filePath),
		FileUpdatedAt: stat.ModTime(),
	}
}

func (f *UploadFile) Name() string {
	return f.RelativePath
}

func (f *UploadFile) Run(ctx context.Context) {
	f.status = tasks.TaskStatusInProgress

	f.watchUpload() // This may dangle if upload will screw up.

	if _, ok := f.Client.PinnedHeader.Files[f.RelativePath]; ok {
		f.UpdateFile(ctx)
		return
	}

	// TODO: update path for encrypted file.
	filePath := f.Client.AbsPath(f.RelativePath)
	stat, _ := os.Stat(filePath)
	// TODO: extract file creation and add encryption data there
	fileInfo := manager.File{
		Name:          filepath.Base(f.RelativePath),
		Path:          f.RelativePath,
		Size:          stat.Size(),
		UploadedAt:    time.Now(),
		FileUpdatedAt: stat.ModTime(),
	}

	d, _ := manager.Marshal(fileInfo)

	msg, err := f.Client.SendMessage(f.Client.chatID, 0, 0,
		tdlib.NewMessageSendOptions(true, false, nil),
		nil,
		tdlib.NewInputMessageDocument(
			tdlib.NewInputFileLocal(filePath),
			nil,
			true,
			tdlib.NewFormattedText(string(d), nil),
		),
	)
	if err != nil {
		f.SetError(fmt.Errorf("upload file: %w", err))
		return
	}

	f.Client.PinnedHeader.Files[f.RelativePath] = msg.ID
	f.Client.FileTree.Add(f.RelativePath, &manager.Tree{File: &fileInfo})
	if err := f.Client.SendHeader(ctx); err != nil {
		f.SetError(err)
		return
	}

	f.SetDone()
}

func (f *UploadFile) UpdateFile(_ context.Context) {
	f.status = tasks.TaskStatusInProgress

	msgID, ok := f.Client.PinnedHeader.Files[f.RelativePath]
	if !ok {
		f.SetError(fmt.Errorf("file not present in the header: %q", f.RelativePath))
		return
	}

	file, _ := manager.FindInTree[*manager.File](f.Client.FileTree, f.RelativePath)

	filePath := f.Client.AbsPath(f.RelativePath)
	stat, _ := os.Stat(filePath)
	file.Size = stat.Size()
	file.FileUpdatedAt = stat.ModTime()
	// TODO: add encryption

	d, _ := manager.Marshal(file)

	_, err := f.Client.TDClient.EditMessageMedia(f.Client.chatID, msgID, nil,
		tdlib.NewInputMessageDocument(
			tdlib.NewInputFileLocal(filePath),
			nil,
			false,
			tdlib.NewFormattedText(string(d), nil),
		),
	)
	if err != nil {
		f.SetError(fmt.Errorf("upload file: %w", err))
		return
	}
	// Header does not need to be updated after update of a file.
	f.SetDone()
}

func (f *UploadFile) watchUpload() {
	var updateState tdlib.UpdateFile
	absFilePath := f.Client.AbsPath(f.RelativePath)

	f.Client.AddUpdateHandler(func(update tdlib.UpdateMsg) bool {
		if update.Data["@type"] != string(tdlib.UpdateFileType) {
			return false
		}

		if err := json.Unmarshal(update.Raw, &updateState); err != nil {
			f.SetError(err)
		}

		if updateState.File.Local.Path != absFilePath {
			return false
		}

		f.progress = int(100 * (updateState.File.Remote.UploadedSize / updateState.File.ExpectedSize))

		return updateState.File.Remote.IsUploadingCompleted
	})
}
