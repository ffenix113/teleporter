package afero_tg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/spf13/afero"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/manager/arman92"
)

var _ afero.Fs = (*Telegram)(nil)

type User struct {
	bun.BaseModel `bun:"table:users"`
	ID            string
	Password      sql.NullString
	ChatName      sql.NullString
	ChatID        sql.NullInt64
}

type Telegram struct {
	cc ftpserver.ClientContext

	userID   string
	chatID   int64
	db       *bun.DB
	tgClient *arman92.Client

	logger *zap.Logger

	Now func() time.Time
}

func NewID() string {
	return uuid.New().String()
}

func NewTelegram(cc ftpserver.ClientContext, userID, password string, client *bun.DB, tgClient *arman92.Client, logger *zap.Logger) (*Telegram, error) {
	var user User
	if err := client.NewSelect().Model(&user).Where("id = ?", userID).
		Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	if user.Password.Valid && bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(password)) != nil {
		return nil, errors.New("invalid password")
	}

	tg := &Telegram{
		cc:       cc,
		userID:   userID,
		db:       client,
		tgClient: tgClient,
		logger:   logger,
		Now:      time.Now,
	}

	if tgClient != nil {
		chat, err := tgClient.FindChat(context.Background(), config.Telegram{ChatName: user.ChatName.String, ChatID: user.ChatID.Int64})
		if err != nil {
			return nil, fmt.Errorf("find chat: %w", err)
		}

		if !chat.Permissions.CanSendMediaMessages {
			return nil, fmt.Errorf("client in chat %q is not allowed to send media messages", chat.Title)
		}

		tg.chatID = chat.ID
	}

	return tg, nil
}

func (t *Telegram) Create(name string) (afero.File, error) {
	// TODO implement me
	panic("implement me Create")
}

func (t *Telegram) Mkdir(name string, perm os.FileMode) error {
	if name == "" {
		name = "/"
	}

	name = filepath.Clean(name)

	if name != "/" {
		base := filepath.Dir(name)
		fileInfo, err := t.fetchItemInfo(context.Background(), base)
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("parent directory does not exist: %q: %w", base, err)
		}

		if err != nil {
			return fmt.Errorf("find file info: %w", err)
		}

		if !fileInfo.IsDir() {
			return fmt.Errorf("%q is not a directory", base)
		}
	}

	now := t.Now()
	dirPath, dirName := filepath.Split(name)
	_, err := t.db.NewInsert().
		Model(&DBFileInfo{
			ID:           NewID(),
			UserID:       t.userID,
			Path:         filepath.Clean(dirPath),
			NameField:    dirName,
			ModTimeField: now,
			ModeField:    perm | fs.ModeDir,
			IsDirField:   true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return nil
}

func (t *Telegram) MkdirAll(dirPath string, perm os.FileMode) error {
	dirPath = filepath.Clean(dirPath)
	searchSuffix := dirPath
	var splitIdx int
	for splitIdx <= len(dirPath) {
		idx := strings.IndexByte(searchSuffix, '/')
		if idx == -1 {
			idx = len(searchSuffix)
		}

		if err := t.Mkdir(dirPath[:splitIdx+idx], perm); err != nil {
			var pgxErr *pgconn.PgError
			if !errors.As(err, &pgxErr) || pgxErr.Code != "23505" {
				return fmt.Errorf("create parent directory: %w", err)
			}
		}

		splitIdx += idx + 1
		if splitIdx <= len(dirPath) {
			searchSuffix = searchSuffix[idx+1:]
		}
	}

	return nil
}

func (t *Telegram) Open(name string) (afero.File, error) {
	return t.OpenFile(name, os.O_RDWR, os.ModePerm)
}

func (t *Telegram) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	dbFile, err := t.fetchItemInfo(context.Background(), name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && flag&os.O_CREATE == os.O_CREATE {
			newFile, err := t.createFile(name, flag, perm)
			if err != nil {
				return nil, fmt.Errorf("create new file: %w", err)
			}

			return newFile, nil
		}

		return nil, err
	}

	if !dbFile.IsDir() {
		return DBFilesInfo{&dbFile}.File(t, flag)
	}

	dbFiles, err := t.fetchItemsInDir(context.Background(), name)
	if err != nil {
		return nil, err
	}

	return &Directory{stat: dbFile, files: dbFiles}, nil
}

func (t *Telegram) Remove(name string) error {
	if name == "/" {
		return fs.ErrInvalid
	}

	fInfo, err := t.fetchItemInfo(context.Background(), name)
	if err != nil {
		return fmt.Errorf("failed to find file info: %w", err)
	}

	if fInfo.IsDir() {
		exists, err := t.dirHasItems(context.Background(), name)
		if err != nil {
			return fmt.Errorf("dir item exists")
		}

		if exists {
			return fmt.Errorf("cannot remove non-empty dir: %s", name)
		}
	} else {
		if err := arman92.DeleteFile(t.tgClient, fInfo.ChatID, fInfo.MessageID); err != nil {
			return fmt.Errorf("delete file: %w", err)
		}
	}

	dirBase, dirName := filepath.Split(name)
	_, err = t.db.NewDelete().
		Model(&fInfo).
		Where("file_path = ? and file_name = ?", filepath.Clean(dirBase), dirName).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("delete path: %w", err)
	}

	return nil
}

func (t *Telegram) RemoveAll(path string) error {
	// TODO implement me
	panic("implement me RemoveAll")
}

func (t *Telegram) Rename(oldname, newname string) error {
	oldFile, oldErr := t.fetchItemInfo(context.Background(), oldname)
	if oldErr != nil {
		return fmt.Errorf("rename: %w", oldErr)
	}
	// Check that newname doesn't exist
	_, newErr := t.fetchItemInfo(context.Background(), newname)
	if newErr != nil && !errors.Is(newErr, fs.ErrNotExist) {
		return fmt.Errorf("rename: %w", newErr)
	}

	// Check that directory for newname exists
	_, dirErr := t.fetchItemInfo(context.Background(), filepath.Dir(newname))
	if dirErr != nil {
		return fmt.Errorf("rename: %w", dirErr)
	}

	if !oldFile.IsDir() {
		// If not directory also need to update Telegram message.
		if err := t.tgClient.ChangeFileCaption(oldFile.ChatID, oldFile.MessageID, newname); err != nil {
			return fmt.Errorf("rename: %w", err)
		}
	}

	newDir, newName := filepath.Split(newname)
	_, err := t.db.NewUpdate().
		Model(&DBFileInfo{}).
		Where("id = ?", oldFile.ID).
		Set("file_path = ?", filepath.Clean(newDir)).
		Set("file_name = ?", newName).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

func (t *Telegram) Stat(name string) (os.FileInfo, error) {
	file, err := t.fetchItemInfo(context.Background(), name)
	if errors.Is(err, sql.ErrNoRows) {
		if name == "/" {
			return nil, nil
		}

		return nil, os.ErrNotExist
	}

	if err != nil {
		return nil, fmt.Errorf("scan file list: %w", err)
	}

	return file, nil
}

func (t *Telegram) Name() string {
	return "telegram"
}

var ErrNotSupported = errors.New("not supported")

func (t *Telegram) Chmod(name string, mode os.FileMode) error {
	return ErrNotSupported
}

func (t *Telegram) Chown(name string, uid, gid int) error {
	return ErrNotSupported
}

func (t *Telegram) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return ErrNotSupported
}

// fetchItemInfo finds file/directory info by path.
func (t *Telegram) fetchItemInfo(ctx context.Context, name string) (DBFileInfo, error) {
	dir, filePath := filepath.Split(name)

	var file DBFileInfo
	if err := t.db.NewSelect().Model(&file).
		Where("user_id = ?", t.userID).
		Where("file_path = ? and file_name = ?", filepath.Clean(dir), filePath).
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DBFileInfo{}, os.ErrNotExist
		}

		return DBFileInfo{}, fmt.Errorf("scan file info: %w", err)
	}

	return file, nil
}

// fetchItemsInDir will return list of files and directories present
// in directory 'name'
func (t *Telegram) fetchItemsInDir(ctx context.Context, name string) (DBFilesInfo, error) {
	var files DBFilesInfo
	if err := t.db.NewSelect().Model(&files).
		Where("user_id = ?", t.userID).
		Where("file_path = ? and file_name != ''", name).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fs.ErrNotExist
		}

		return nil, fmt.Errorf("scan file list: %w", err)
	}

	return files, nil
}

func (t *Telegram) dirHasItems(ctx context.Context, name string) (bool, error) {
	exists, err := t.db.NewSelect().Model(&DBFileInfo{}).
		Where("user_id = ?", t.userID).
		Where("file_path = ?", name).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("scan item list: %w", err)
	}

	return exists, nil
}

func (t *Telegram) createFile(name string, flag int, perm fs.FileMode) (*File, error) {
	fileDir, fileName := filepath.Split(name)
	now := time.Now()

	tmpFile, err := os.CreateTemp(t.tgClient.TempPath, "*_"+fileName)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	dbFile := &DBFileInfo{
		ID:           NewID(),
		UserID:       t.userID,
		ChatID:       t.chatID,
		Path:         filepath.Clean(fileDir),
		NameField:    fileName,
		ModeField:    perm,
		ModTimeField: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	file := &File{
		driver: t,
		File:   tmpFile,
		flag:   flag,
		files:  DBFilesInfo{dbFile},
	}

	return file, nil
}

func (t *Telegram) insertFile(ctx context.Context, file *File) error {
	dbFile := file.files[0]

	_, err := t.db.NewInsert().
		Model(dbFile).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	return nil
}

func (t *Telegram) updateFile(ctx context.Context, file *File) error {
	dbFile := file.files[0]

	_, err := t.db.NewUpdate().
		Model(dbFile).
		WherePK().
		Set("file_id = ?", dbFile.FileID).
		Set("size = ?", dbFile.SizeField).
		Set("updated_at = ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	return nil
}

var _flagBits = map[int]string{
	os.O_RDONLY: "O_RDONLY",
	os.O_RDWR:   "O_RDWR",
	os.O_WRONLY: "O_WRONLY",
	os.O_APPEND: "O_APPEND",
	os.O_CREATE: "O_CREATE",
	os.O_EXCL:   "O_EXCL",
	os.O_SYNC:   "O_SYNC",
	os.O_TRUNC:  "O_TRUNC",
}

func flagBits(flag int) (bts []string) {
	for flagBit, flagName := range _flagBits {
		if flag&flagBit == flagBit {
			bts = append(bts, flagName)
		}
	}

	return
}
