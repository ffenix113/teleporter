package afero_tg

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/afero"

	"github.com/ffenix113/teleporter/manager/arman92"
)

type RemoteFileReader struct {
	Client    *arman92.Client
	FileID    int32
	offset    int32
	chunkSize int32
	finished  bool

	afero.File
}

func NewRemoteFileReader(client *arman92.Client, fileID int32, chunkSize int) *RemoteFileReader {
	return &RemoteFileReader{
		Client:    client,
		FileID:    fileID,
		chunkSize: int32(chunkSize),
	}
}

func (r *RemoteFileReader) Read(b []byte) (int, error) {
	if r.finished {
		return 0, io.EOF
	}

	fl, err := r.Client.TDClient.DownloadFile(r.FileID, 1, r.offset, r.chunkSize, true)
	if err != nil {
		return 0, fmt.Errorf("download file: %w", err)
	}

	if r.File == nil {
		r.File, err = os.Open(fl.Local.Path)
		if err != nil {
			return 0, fmt.Errorf("open file: %w", err)
		}
		log.Printf("opened remote file: %q\n", fl.Local.Path)
	}

	if len(b) > int(r.chunkSize) {
		b = b[:r.chunkSize]
	}

	r.finished = fl.Local.IsDownloadingCompleted
	r.offset += int32(len(b))

	return r.File.Read(b)
}
