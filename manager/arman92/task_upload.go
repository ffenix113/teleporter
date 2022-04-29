package arman92

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Arman92/go-tdlib/v2/tdlib"
)

type UploadFile struct {
	*Common
	TempPath, RealPath string
	FileUpdatedAt      time.Time
}

func NewUploadFile(cl *Client, tempFilePath, realPath string) *UploadFile {
	return &UploadFile{
		Common: &Common{
			Client:   cl,
			taskType: "UploadFile",
		},

		TempPath: tempFilePath,
		RealPath: realPath,
	}
}

func (f *UploadFile) Upload(ctx context.Context, chatID int64) (msgID int64, fileID int32, err error) {
	uploadFinished := f.watchUpload() // This may dangle if upload will screw up.

	// TODO: update path for encrypted file.
	msg, err := f.Client.SendMessage(chatID, 0, 0,
		tdlib.NewMessageSendOptions(true, true, nil),
		nil,
		tdlib.NewInputMessageDocument(
			tdlib.NewInputFileLocal(f.TempPath),
			nil,
			true,
			tdlib.NewFormattedText(f.RealPath, nil),
		),
	)

	if err != nil {
		return 0, 0, fmt.Errorf("upload file: %w", err)
	}

	msgDoc, ok := msg.Content.(*tdlib.MessageDocument)
	if !ok {
		return 0, 0, fmt.Errorf("message is not document: %v", msg.Content)
	}

	<-uploadFinished

	return msg.ID, msgDoc.Document.Document.ID, nil
}

func (f *UploadFile) Update(_ context.Context, chatID, msgID int64) (int32, error) {
	uploadFinished := f.watchUpload() // This may dangle if upload will screw up.

	msg, err := f.Client.TDClient.EditMessageMedia(chatID, msgID, nil,
		tdlib.NewInputMessageDocument(
			tdlib.NewInputFileLocal(f.TempPath),
			nil,
			false,
			tdlib.NewFormattedText(f.RealPath, nil),
		),
	)
	if err != nil {
		return 0, fmt.Errorf("upload file: %w", err)
	}

	msgDoc, ok := msg.Content.(*tdlib.MessageDocument)
	if !ok {
		return 0, fmt.Errorf("message is not document: %v", msg.Content)
	}

	<-uploadFinished

	return msgDoc.Document.Document.ID, nil
}

func (f *UploadFile) watchUpload() <-chan struct{} {
	var updateState tdlib.UpdateFile

	updChn := make(chan struct{})

	f.Client.AddUpdateHandler(func(update tdlib.UpdateMsg) bool {
		if update.Data["@type"] != string(tdlib.UpdateFileType) {
			return false
		}

		if err := json.Unmarshal(update.Raw, &updateState); err != nil {
			f.SetError(err)
		}

		if updateState.File.Local.Path != f.TempPath {
			return false
		}

		if updateState.File.Remote.IsUploadingCompleted {
			close(updChn)
		}

		return updateState.File.Remote.IsUploadingCompleted
	})

	return updChn
}
