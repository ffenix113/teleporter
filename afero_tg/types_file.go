package afero_tg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

type File struct {
	*os.File
	driver *Telegram

	modifiedAt time.Time
	// flag is the flags that are passed in to OpenFile/Create
	flag  int
	files DBFilesInfo
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	return nil, nil
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
