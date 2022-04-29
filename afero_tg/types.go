package afero_tg

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"time"

	"github.com/Arman92/go-tdlib/v2/tdlib"
	"github.com/spf13/afero"
	"github.com/uptrace/bun"
)

var _ fs.FileInfo = DBFileInfo{}

const DownloadChunkSize = 16 * 1024

type DBFilesInfo []*DBFileInfo

func (i DBFilesInfo) File(driver *Telegram, flag int) (afero.File, error) {
	if len(i) == 1 && !i[0].IsDir() {
		fl := i[0]

		if err := driver.tgClient.EnsureMessagesAreKnown(context.Background(), fl.ChatID, fl.MessageID); err != nil {
			return nil, fmt.Errorf("failed to ensure message is known: %w", err)
		}

		msg, err := driver.tgClient.TDClient.GetMessage(driver.chatID, fl.MessageID)
		if err != nil {
			// TODO: remove file from DB when message is deleted.
			var reqErr tdlib.RequestError
			if errors.As(err, &reqErr) && reqErr.Code == 404 {
				if err := driver.Remove(fl.AbsName()); err != nil {
					return nil, fmt.Errorf("failed to remove deleted file: %w", err)
				}
			}

			return nil, fmt.Errorf("failed to get message: %w", err)
		}

		remoteReader := NewRemoteFileReader(driver.tgClient, msg.Content.(*tdlib.MessageDocument).Document.Document.ID, DownloadChunkSize)

		return &File{
			driver: driver,
			flag:   flag,
			File:   remoteReader,
			files:  i,
		}, nil
	}

	// This is a directory, just pass in file info that we have
	return &Directory{
		files: i,
	}, nil
}

type DBFileInfo struct {
	bun.BaseModel `bun:"table:files"`

	ID        string
	UserID    string
	ChatID    int64
	MessageID int64
	FileID    int32
	Path      string `bun:"file_path"`

	NameField    string      `bun:"file_name"`
	SizeField    int64       `bun:"size"`
	ModeField    fs.FileMode `bun:"file_mode"`
	ModTimeField time.Time   `bun:"mod_time"`
	IsDirField   bool        `bun:"is_dir"`

	Metadata map[string]interface{} `bun:"type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (f DBFileInfo) Name() string {
	// While normal behavior is to return name as it was provided
	// in Open/Create/etc call - here we will always return
	// relative path to the item, as clients break if we
	// don't do this.
	return f.NameField
}

func (f DBFileInfo) AbsName() string {
	return path.Join(f.Path, f.NameField)
}

func (f DBFileInfo) Size() int64 {
	return f.SizeField
}

func (f DBFileInfo) Mode() fs.FileMode {
	return f.ModeField
}

func (f DBFileInfo) ModTime() time.Time {
	return f.ModTimeField
}

func (f DBFileInfo) IsDir() bool {
	return f.IsDirField
}

func (f DBFileInfo) Sys() any {
	return nil
}
