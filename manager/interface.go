package manager

import (
	"context"
	"time"
)

type File struct {
	Path      string
	UpdatedAt time.Time
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
