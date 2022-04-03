package manager

import (
	"context"
	"time"
)

type File struct {
	// Name specifies name of the file or directory
	Name string `json:",omitempty"`
	// Path specifies where current file or directory is located
	// For root directory, path is empty string.
	Path string `json:",omitempty"`
	// Size specifies size of the file.
	Size          int64     `json:",omitempty"`
	UploadedAt    time.Time `json:",omitempty"`
	FileUpdatedAt time.Time `json:",omitempty"`

	IsDir bool `json:",omitempty"`
	// Encrypted is a base64 of encrypted fields above.
	Encrypted string `json:",omitempty"`
}

type PinnedHeader struct {
	Header string           // Constant value to be able to search for this message
	Files  map[string]int64 `json:",omitempty"` // Map filepath -> messageID
	// Encrypted is a base64 of encrypted fields above.
	Encrypted string `json:",omitempty"`
}

type Manager interface {
	ListAllFiles() []*File
	GetFile(ctx context.Context, path string) ([]byte, error)
}
