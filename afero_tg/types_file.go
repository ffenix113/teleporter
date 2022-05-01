package afero_tg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/afero"
)

type File struct {
	afero.File
	driver *Telegram

	modifiedAt time.Time
	// flag is the flags that are passed in to OpenFile/Create
	flag  int
	files DBFilesInfo
}

func (f *File) Write(p []byte) (n int, err error) {
	return f.File.Write(p)
}

func (f *File) ReadFrom(r io.Reader) (n int64, err error) {
	buf := make([]byte, 32*1024)

	for {
		rn, err := r.Read(buf)

		if rn != 0 {
			if _, err := f.Write(buf[:rn]); err != nil {
				return n, err
			}
		}

		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return n, err
		}

		n += int64(rn)
	}
}

func (f *File) Close() error {
	if f.flag == os.O_RDONLY {
		return nil
	}

	if f.flag&os.O_CREATE == os.O_CREATE {
		if err := f.upload(); err != nil {
			return fmt.Errorf("upload: %w", err)
		}

		if err := f.driver.insertFile(context.Background(), f); err != nil {
			return fmt.Errorf("insert file: %w", err)
		}
	} else if f.flag&os.O_WRONLY == os.O_WRONLY || f.flag&os.O_RDWR == os.O_RDWR || f.flag&os.O_APPEND == os.O_APPEND {
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

	stat, _ := f.File.Stat()

	dbFile := f.files[0]
	dbFile.SizeField = stat.Size()
	dbFile.MessageID = messageID
	dbFile.FileID = fileID

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

	return nil
}
