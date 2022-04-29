package afero_tg

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/Arman92/go-tdlib/v2/tdlib"
	"github.com/uptrace/bun"
)

var _ fs.FileInfo = DBFileInfo{}

type DBFilesInfo []*DBFileInfo

func (i DBFilesInfo) File(driver *Telegram, flag int, perm os.FileMode) (*File, error) {
	switch len(i) {
	case 0:
		return &File{
			files:  i,
			flag:   flag,
			driver: driver,
		}, nil
	case 1:
		fl := i[0]
		if fl.IsDir() {
			break
		}

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

		filePath, err := driver.tgClient.EnsureLocalFileExists(msg.Content.(*tdlib.MessageDocument).Document.Document.ID)
		if err != nil {
			return nil, fmt.Errorf("ensure local file: %w", err)
		}

		osFile, err := os.OpenFile(filePath, flag, perm)
		if err != nil {
			return nil, fmt.Errorf("open file: %w", err)
		}

		stat, _ := osFile.Stat()

		return &File{
			driver:     driver,
			flag:       flag,
			File:       osFile,
			modifiedAt: stat.ModTime(),
			files:      i,
		}, nil
	}

	// This is a directory, just pass in file info that we have
	return &File{
		files:  i,
		flag:   flag,
		driver: driver,
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

type File struct {
	*os.File
	driver *Telegram

	name       string
	modifiedAt time.Time
	// flag is the flags that are passed in to OpenFile/Create
	flag  int
	stat  fs.FileInfo
	files DBFilesInfo

	readDirN      int
	readDirNamesN int
}

func (f *File) Name() string {
	if f.File != nil {
		return f.File.Name()
	}

	if f.stat != nil {
		return f.stat.Name()
	}

	return f.name
}

func (f *File) Stat() (fs.FileInfo, error) {
	if f.File != nil {
		return f.File.Stat()
	}

	return f.stat, nil
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	if count == -1 {
		count = len(f.files)
	}
	if count > len(f.files)-f.readDirN {
		count = len(f.files) - f.readDirN
	}
	if count < 0 {
		return nil, nil
	}

	var files []os.FileInfo
	for _, file := range f.files[f.readDirN : f.readDirN+count] {
		files = append(files, file)
	}

	f.readDirN += count

	return files, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	if n == -1 {
		n = len(f.files)
	}
	if n > len(f.files)-f.readDirNamesN {
		n = len(f.files) - f.readDirNamesN
	}
	if n < 0 {
		return nil, nil
	}

	var dirNames []string

	for _, file := range f.files[f.readDirNamesN : f.readDirNamesN+n] {
		if file.IsDirField {
			dirNames = append(dirNames, file.NameField)
		}
	}

	f.readDirNamesN += n

	return dirNames, nil
}

func (f *File) Sync() error {
	if f.File != nil {
		return f.File.Sync()
	}

	return nil
}

func (f *File) Close() error {
	if f.File == nil {
		return nil
	}

	stat, _ := f.File.Stat()
	dbFile := f.files[0]
	dbFile.SizeField = stat.Size()

	if f.flag&os.O_CREATE != 0 {
		if err := f.upload(); err != nil {
			return fmt.Errorf("upload: %w", err)
		}

		if err := f.driver.insertFile(context.Background(), f); err != nil {
			return fmt.Errorf("insert file: %w", err)
		}
	} else {
		stat, _ := f.File.Stat()
		if stat.ModTime().After(f.modifiedAt) {
			if err := f.update(); err != nil {
				return fmt.Errorf("upload: %w", err)
			}

			if err := f.driver.updateFile(context.Background(), f); err != nil {
				return fmt.Errorf("insert file: %w", err)
			}
		}
	}

	if err := os.Remove(f.File.Name()); err != nil {
		return fmt.Errorf("remove temp file: %w", err)
	}

	return f.File.Close()
}

func (f *File) upload() error {
	if f.File == nil {
		return errors.New("trying to upload without a file")
	}

	messageID, fileID, err := f.driver.tgClient.UploadFile(f.driver.chatID, f.File.Name(), f.files[0].AbsName())
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	dbFile := f.files[0]
	dbFile.MessageID = messageID
	dbFile.FileID = fileID

	if _, err := f.driver.tgClient.TDClient.DeleteFile(fileID); err != nil {
		return fmt.Errorf("delete file by tdclient: %w", err)
	}

	return nil
}
func (f *File) update() error {
	if f.File == nil {
		return nil
	}

	fileID, err := f.driver.tgClient.UpdateFile(f.files[0].ChatID, f.files[0].MessageID, f.File.Name(), f.files[0].AbsName())
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	f.files[0].FileID = fileID

	if _, err := f.driver.tgClient.TDClient.DeleteFile(fileID); err != nil {
		return fmt.Errorf("delete file by tdclient: %w", err)
	}

	return nil
}
