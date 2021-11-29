package arman92

import (
	"context"
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Arman92/go-tdlib"
	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/tasks"
	"gopkg.in/yaml.v3"
)

type Common struct {
	Client   *Client
	taskType string
	status   tasks.TaskStatus
	progress int
	details  string
}

func NewCommon(cl *Client, taskType string) *Common {
	return &Common{Client: cl, taskType: taskType}
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
func NewDownloadFile(cl *Client, filePath string) *DownloadFile {
	return &DownloadFile{
		Common: &Common{
			Client:   cl,
			taskType: "DownloadFile",
		},
		RelativePath: cl.RelativePath(filePath),
	}
}

func (f *DownloadFile) Name() string {
	return f.RelativePath
}

func (f *DownloadFile) Run(ctx context.Context) {
	f.status = tasks.TaskStatusInProgress

	msgID, ok := f.Client.filesHeader.Files[f.RelativePath]
	if !ok {
		f.SetError(fmt.Errorf("file %q is not present in remote chat", f.RelativePath))
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

	// Maybe would need to update offset and limit on file DL update.
	file, err := f.Client.Client.DownloadFile(msgDoc.Document.Document.ID, 0, 0, 0, true)
	if err != nil {
		f.SetError(err)
		return
	}

	for !file.Local.IsDownloadingCompleted {
		time.Sleep(200 * time.Millisecond)
	}

	if err := os.MkdirAll(path.Dir(f.Client.AbsPath(f.RelativePath)), os.ModeDir); err != nil {
		f.SetError(err)
		return
	}

	if err := os.Rename(file.Local.Path, f.Client.AbsPath(f.RelativePath)); err != nil {
		f.SetError(fmt.Errorf("move file: %w", err))
	}

	f.SetDone()
}

type UploadFile struct {
	*Common
	RelativePath string
}

func NewUploadFile(cl *Client, filePath string) *UploadFile {
	return &UploadFile{
		Common: &Common{
			Client:   cl,
			taskType: "UploadFile",
		},
		RelativePath: cl.RelativePath(filePath),
	}
}

func (f *UploadFile) Name() string {
	return f.RelativePath
}

func (f *UploadFile) Run(ctx context.Context) {
	f.status = tasks.TaskStatusInProgress

	if _, ok := f.Client.filesHeader.Files[f.RelativePath]; ok {
		f.UpdateFile(ctx)
		return
	}

	fInfo, err := os.Stat(f.Client.AbsPath(f.RelativePath))
	if err != nil {
		f.SetError(err)
		return
	}

	fileInfo := manager.File{
		Path:      f.RelativePath,
		Size:      fInfo.Size(),
		UpdatedAt: time.Now(),
		Encrypted: "",
	}

	d, _ := yaml.Marshal(fileInfo)

	msg, err := f.Client.Client.SendMessage(f.Client.chatID, 0, 0,
		tdlib.NewMessageSendOptions(true, false, nil),
		nil,
		tdlib.NewInputMessageDocument(
			tdlib.NewInputFileLocal(f.Client.AbsPath(f.RelativePath)),
			nil,
			false,
			tdlib.NewFormattedText(string(d), nil),
		),
	)
	if err != nil {
		f.SetError(fmt.Errorf("upload file: %w", err))
		return
	}

	f.Client.filesHeader.Files[f.RelativePath] = msg.ID
	if err := f.Client.SendHeader(ctx); err != nil {
		f.SetError(err)
		return
	}

	f.SetDone()
}

func (f *UploadFile) UpdateFile(ctx context.Context) {
	f.status = tasks.TaskStatusInProgress

	msgID, ok := f.Client.filesHeader.Files[f.RelativePath]
	if !ok {
		f.SetError(fmt.Errorf("file not present in the header: %q", f.RelativePath))
		return
	}

	fInfo, err := os.Stat(f.Client.AbsPath(f.RelativePath))
	if err != nil {
		f.SetError(err)
		return
	}

	fileInfo := manager.File{
		Path:      f.RelativePath,
		Size:      fInfo.Size(),
		UpdatedAt: time.Now(),
	}
	// TODO: add encryption

	d, _ := yaml.Marshal(fileInfo)

	_, err = f.Client.Client.EditMessageMedia(f.Client.chatID, msgID, nil,
		tdlib.NewInputMessageDocument(
			tdlib.NewInputFileLocal(f.Client.AbsPath(f.RelativePath)),
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

	msgID, ok := f.Client.filesHeader.Files[f.RelativePath]
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

	delete(f.Client.filesHeader.Files, f.RelativePath)
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
	for filePath, msgID := range d.Client.filesHeader.Files {
		if !strings.HasPrefix(filePath, d.RelativeDirPath) {
			continue
		}

		delete(d.Client.filesHeader.Files, filePath)
		msgsToDelete = append(msgsToDelete, msgID)
	}
	// Can also be done as a separate tasks to delete provided files
	// by using DeleteFile.
	_, err := d.Client.Client.DeleteMessages(d.Client.chatID, msgsToDelete, true)
	if err != nil {
		d.SetError(err)
		return
	}

	if err := d.Client.SendHeader(ctx); err != nil {
		d.SetError(err)
		return
	}

	d.SetDone()
}
