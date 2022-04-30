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

func (d *Directory) Size() int64 {
	return d.stat.Size()
}

func (d *Directory) Mode() fs.FileMode {
	return d.stat.Mode()
}

func (d *Directory) ModTime() time.Time {
	return d.stat.ModTime()
}

func (d *Directory) IsDir() bool {
	return true
}

func (d *Directory) Sys() any {
	return d.stat.Sys()
}

func (d *Directory) Name() string {
	return d.name
}

func (d *Directory) Stat() (fs.FileInfo, error) {
	return d.stat, nil
}

func (d *Directory) Readdir(count int) ([]os.FileInfo, error) {
	if count == -1 {
		count = len(d.files)
	}
	if count > len(d.files)-d.readDirN {
		count = len(d.files) - d.readDirN
	}
	if count < 0 {
		return nil, nil
	}

	var files []os.FileInfo
	for _, file := range d.files[d.readDirN : d.readDirN+count] {
		files = append(files, file)
	}

	d.readDirN += count

	return files, nil
}

func (d *Directory) Readdirnames(n int) ([]string, error) {
	if n == -1 {
		n = len(d.files)
	}
	if n > len(d.files)-d.readDirNamesN {
		n = len(d.files) - d.readDirNamesN
	}
	if n < 0 {
		return nil, nil
	}

	var dirNames []string

	for _, file := range d.files[d.readDirNamesN : d.readDirNamesN+n] {
		if file.IsDirField {
			dirNames = append(dirNames, file.NameField)
		}
	}

	d.readDirNamesN += n

	return dirNames, nil
}

func (d *Directory) Close() error {
	return nil
}
