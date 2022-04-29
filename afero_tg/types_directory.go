package afero_tg

import (
	"io/fs"
	"os"
	"time"

	"github.com/spf13/afero"
)

var _ afero.File = (*Directory)(nil)

type Directory struct {
	afero.File // Embed so there will be no issues with implementing all methods

	name  string
	stat  fs.FileInfo
	files DBFilesInfo

	readDirN      int
	readDirNamesN int
}

func (f *Directory) Size() int64 {
	return f.stat.Size()
}

func (f *Directory) Mode() fs.FileMode {
	return f.stat.Mode()
}

func (f *Directory) ModTime() time.Time {
	return f.stat.ModTime()
}

func (f *Directory) IsDir() bool {
	return true
}

func (f *Directory) Sys() any {
	return f.stat.Sys()
}

func (f *Directory) Name() string {
	return f.name
}

func (f *Directory) Stat() (fs.FileInfo, error) {
	return f.stat, nil
}

func (f *Directory) Readdir(count int) ([]os.FileInfo, error) {
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

func (f *Directory) Readdirnames(n int) ([]string, error) {
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

func (d *Directory) Close() error {
	return nil
}
