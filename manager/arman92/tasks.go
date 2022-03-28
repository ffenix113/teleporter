package arman92

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Arman92/go-tdlib/v2/tdlib"

	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/tasks"
)

// https://core.telegram.org/api/files
const DownloadFilePartSize = 16 * 1024

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

func (StaticTask) Run(ctx context.Context) {}

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

	msg, err := f.Client.Client.GetMessage(f.Client.chatID, msgID)
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
	if file, err := f.Client.GetFile(fileID); err == nil {
		if !file.Local.IsDownloadingCompleted {
			if _, err := f.Client.CancelDownloadFile(fileID, false); err != nil {
				f.SetError(err)
				return
			}

			watcher := f.watchDownload(fileID) // This may dangle if download will screw up.
			// Download(msgDoc.Document.Document.ID, DownloadFilePartSize)
			_, err = f.Client.DownloadFile(fileID, 1, 0, 0, false)
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
		file, err = f.Client.DownloadFile(fileID, 1, offset, partSize, true)
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

		json.Unmarshal(update.Raw, &fileUpdate)
		if fileUpdate.File.ID != fileID {
			return false
		}

		if fileUpdate.File.Local.IsDownloadingCompleted {
			watcher <- fileUpdate.File
			close(watcher)
			return true
		}

		f.progress = int(100 * (float64(fileUpdate.File.Local.DownloadedSize) / float64(fileUpdate.File.ExpectedSize)))

		return false
	})

	return watcher
}

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

func (f *UploadFile) UpdateFile(ctx context.Context) {
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

	_, err := f.Client.Client.EditMessageMedia(f.Client.chatID, msgID, nil,
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
	// It is safer to have deleted file and entry left in header
	// than the other way around. If header entry will be missing for a file
	// files will be leaking.
	_, err := f.Client.Client.DeleteMessages(f.Client.chatID, []int64{msgID}, true)
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
	_, err := d.Client.Client.DeleteMessages(d.Client.chatID, msgsToDelete, true)
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

func detailsOrEmpty(strs ...string) string {
	if len(strs) == 0 {
		return ""
	}

	return strs[0]
}
