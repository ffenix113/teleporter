package afero_tg

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Arman92/go-tdlib/v2/tdlib"
	"github.com/spf13/afero"

	"github.com/ffenix113/teleporter/manager/arman92"
)

type FileReader struct {
	Client    *arman92.Client
	FileID    int32
	tdFile    *tdlib.File
	offset    int32
	chunkSize int32
	finished  bool

	afero.File
}

// NewFileReader creates a new FileReader.
//
// If file is available locally - it will be provided instead.
func NewFileReader(client *arman92.Client, fileID int32, chunkSize int) (afero.File, error) {
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
		Client:    client,
		FileID:    fileID,
		tdFile:    fl,
		chunkSize: int32(chunkSize),
	}, nil
}

func (r *FileReader) Read(b []byte) (int, error) {
	if r.finished {
		return 0, io.EOF
	}

	if !r.finished {
		var err error
		r.tdFile, err = r.Client.TDClient.DownloadFile(r.FileID, 1, r.offset, r.chunkSize, true)
		if err != nil {
			return 0, fmt.Errorf("download file: %w", err)
		}
	}

	if r.File == nil {
		var err error
		r.File, err = os.Open(r.tdFile.Local.Path)
		if err != nil {
			return 0, fmt.Errorf("open file: %w", err)
		}
		log.Printf("opened remote file: %q\n", r.tdFile.Local.Path)
	}

	if len(b) > int(r.chunkSize) {
		b = b[:r.chunkSize]
	}

	r.finished = r.tdFile.Local.IsDownloadingCompleted
	r.offset += int32(len(b))

	return r.File.Read(b)
}
