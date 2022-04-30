package afero_tg

import (
	"os"
	"time"

	"github.com/spf13/afero"
	"go.uber.org/zap"
)

var _ afero.Fs = Wrapper{}

type Wrapper struct {
	fs     afero.Fs
	logger *zap.Logger
}

func NewWrapper(fs afero.Fs, logger *zap.Logger) afero.Fs {
	return Wrapper{
		fs:     fs,
		logger: logger,
	}
}

func (w Wrapper) Create(name string) (afero.File, error) {
	w.logger.Debug("create", zap.String("name", name))

	return w.fs.Create(name)
}

func (w Wrapper) Mkdir(name string, perm os.FileMode) error {
	w.logger.Debug("mkdir", zap.String("name", name), zap.String("perm", perm.String()))

	return w.fs.Mkdir(name, perm)
}

func (w Wrapper) MkdirAll(path string, perm os.FileMode) error {
	w.logger.Debug("mkdirall", zap.String("path", path), zap.String("perm", perm.String()))

	return w.fs.MkdirAll(path, perm)
}

func (w Wrapper) Open(name string) (afero.File, error) {
	w.logger.Debug("open", zap.String("name", name))

	return w.fs.Open(name)
}

func (w Wrapper) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	w.logger.Debug("openfile", zap.String("name", name), zap.Strings("flag", flagBits(flag)), zap.String("perm", perm.String()))

	return w.fs.OpenFile(name, flag, perm)
}

func (w Wrapper) Remove(name string) error {
	w.logger.Debug("remove", zap.String("name", name))

	return w.fs.Remove(name)
}

func (w Wrapper) RemoveAll(path string) error {
	w.logger.Debug("removeall", zap.String("path", path))

	return w.fs.RemoveAll(path)
}

func (w Wrapper) Rename(oldname, newname string) error {
	w.logger.Debug("rename", zap.String("oldname", oldname), zap.String("newname", newname))

	return w.fs.Rename(oldname, newname)
}

func (w Wrapper) Stat(name string) (os.FileInfo, error) {
	w.logger.Debug("stat", zap.String("name", name))

	return w.fs.Stat(name)
}

func (w Wrapper) Name() string {
	return w.fs.Name()
}

func (w Wrapper) Chmod(name string, mode os.FileMode) error {
	w.logger.Debug("chmod", zap.String("name", name), zap.String("mode", mode.String()))

	return w.fs.Chmod(name, mode)
}

func (w Wrapper) Chown(name string, uid, gid int) error {
	w.logger.Debug("chown", zap.String("name", name), zap.Int("uid", uid), zap.Int("gid", gid))

	return w.fs.Chown(name, uid, gid)
}

func (w Wrapper) Chtimes(name string, atime time.Time, mtime time.Time) error {
	w.logger.Debug("chtimes", zap.String("name", name), zap.Time("atime", atime), zap.Time("mtime", mtime))

	return w.fs.Chtimes(name, atime, mtime)
}
