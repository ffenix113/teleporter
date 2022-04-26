package ftp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"log"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/jackc/pgconn"
	"github.com/uptrace/bun"
	"go.uber.org/zap"

	"github.com/ffenix113/teleporter/afero"
	"github.com/ffenix113/teleporter/manager/arman92"
)

var _ ftpserver.MainDriver = (*Driver)(nil)

type Driver struct {
	db       *bun.DB
	tgClient *arman92.Client

	settings *ftpserver.Settings

	logger *zap.Logger
}

func NewDriver(db *bun.DB, tgClient *arman92.Client, settings *ftpserver.Settings, logger *zap.Logger) *Driver {
	return &Driver{
		db:       db,
		tgClient: tgClient,
		settings: settings,
		logger:   logger,
	}
}

func (d *Driver) GetSettings() (*ftpserver.Settings, error) {
	return d.settings, nil
}

func (d *Driver) ClientConnected(cc ftpserver.ClientContext) (string, error) {
	cc.SetDebug(true)
	log.Printf("New connection from %q, client id: %d\n", cc.RemoteAddr(), cc.ID())

	return "Teleporter", nil
}

func (d *Driver) ClientDisconnected(cc ftpserver.ClientContext) {}

func (d *Driver) AuthUser(cc ftpserver.ClientContext, user, pass string) (ftpserver.ClientDriver, error) {
	logger := d.logger.With(zap.String("user", user))

	driver, err := afero.NewTelegram(cc, user, d.db, d.tgClient, logger)
	if err != nil {
		return nil, fmt.Errorf("create driver: %w", err)
	}

	tg := afero.NewWrapper(driver, logger)

	d.logger.Debug("creating root")
	if err := tg.MkdirAll("/", fs.ModePerm|fs.ModeDir); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code != "23505" {
				return nil, fmt.Errorf("create root: %w", err)
			}
		}
	}

	return tg, nil
}

func (d *Driver) GetTLSConfig() (*tls.Config, error) {
	return nil, nil
}
