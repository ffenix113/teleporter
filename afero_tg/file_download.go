package afero_tg

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/Arman92/go-tdlib/v2/tdlib"
	"github.com/spf13/afero"

	"github.com/ffenix113/teleporter/manager/arman92"
)

type FileReader struct {
	Client *arman92.Client
	FileID int32
	tdFile *tdlib.File
	init   sync.Once
	read   int32
	notify chan int32

	afero.File
}

// NewFileReader creates a new FileReader.
//
// If file is available locally - it will be provided instead.
func NewFileReader(client *arman92.Client, fileID int32) (afero.File, error) {
	fl, err := client.TDClient.GetFile(fileID)
	if err != nil {
		return nil, fmt.Errorf("remote file: %w", err)
	}

	if fl.Local.IsDownloadingCompleted {
		return os.Open(fl.Local.Path)
	}

	if !fl.Local.CanBeDownloaded {
		return nil, fmt.Errorf("remote file %d can't be downloaded", fileID)
	}

	return &FileReader{
		Client: client,
		FileID: fileID,
		tdFile: fl,
		notify: make(chan int32),
	}, nil
}

func (r *FileReader) Read(b []byte) (n int, err error) {
	r.init.Do(func() {
		// FIXME: this will allow only one user at a time to download file.
		//  if multiple users will download the same file - it will break.
		r.tdFile, err = r.Client.TDClient.DownloadFile(r.FileID, 1, 0, 0, false)
		if err != nil {
			return
		}

		r.Client.AddUpdateHandler(r.downloadFileHandler)

		r.File, err = os.Open(r.tdFile.Local.Path)
		if err != nil {
			return
		}
	})

	newAvailable, ok := <-r.notify
	if ok {
		available := newAvailable - r.read
		if len(b) > int(available) {
			b = b[:available]
		}
	}

	n, err = r.File.Read(b)
	r.read += int32(n)

	return n, err
}

func (r *FileReader) Close() error {
	_, err := r.Client.TDClient.CancelDownloadFile(r.FileID, false)

	if r.File != nil {
		return r.File.Close()
	}

	return err
}

func (r *FileReader) downloadFileHandler(update tdlib.UpdateMsg) bool {
	if update.Data["@type"].(string) != string(tdlib.UpdateFileType) {
		return false
	}

	var updateFile tdlib.UpdateFile
	if err := json.Unmarshal(update.Raw, &updateFile); err != nil {
		panic(err)
	}

	file := updateFile.File

	if file.ID != r.FileID {
		return false
	}

	select {
	case r.notify <- file.Local.DownloadedPrefixSize:
	default: // Drop if previous value was not yet read.
	}

	downloadFinished := file.Local.IsDownloadingCompleted || !file.Local.IsDownloadingActive
	if downloadFinished {
		close(r.notify)
	}

	return downloadFinished
}
