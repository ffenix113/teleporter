package ftp

import (
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/jackc/pgconn"
	"github.com/spf13/afero"
	"github.com/uptrace/bun"
	"go.uber.org/zap"

	"github.com/ffenix113/teleporter/afero_tg"
	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/manager/arman92"
)

var _ ftpserver.MainDriver = (*Driver)(nil)

type Driver struct {
	db       *bun.DB
	tgClient *arman92.Client

	settings config.FTP

	logger *zap.Logger
}

func NewDriver(db *bun.DB, tgClient *arman92.Client, settings config.FTP, logger *zap.Logger) *Driver {
	return &Driver{
		db:       db,
		tgClient: tgClient,
		settings: settings,
		logger:   logger,
	}
}

func (d *Driver) GetSettings() (*ftpserver.Settings, error) {
	return d.settings.Settings, nil
}

func (d *Driver) ClientConnected(cc ftpserver.ClientContext) (string, error) {
	cc.SetDebug(d.settings.Debug)
	log.Printf("New connection from %q, client id: %d\n", cc.RemoteAddr(), cc.ID())

	return "Teleporter", nil
}

func (d *Driver) ClientDisconnected(cc ftpserver.ClientContext) {}

func (d *Driver) AuthUser(cc ftpserver.ClientContext, user, pass string) (ftpserver.ClientDriver, error) {
	host, _, err := net.SplitHostPort(cc.RemoteAddr().String())
	if err != nil {
		return nil, fmt.Errorf("cannot get client host from: %q", cc.RemoteAddr())
	}

	if _, ok := d.settings.IPWhitelistMap[host]; len(d.settings.IPWhitelistMap) != 0 && !ok {
		return nil, fmt.Errorf("client %q is not allowed to connect", host)
	}

	if len(d.settings.Users) != 0 {
		password, ok := d.settings.Users[user]
		if !ok {
			return nil, fmt.Errorf("user %q not found", user)
		}

		if user != "anonymous" && subtle.ConstantTimeCompare([]byte(password), []byte(pass)) != 1 {
			return nil, fmt.Errorf("wrong password")
		}
	}

	logger := d.logger.With(zap.String("user", user))

	driver, err := afero_tg.NewTelegram(cc, user, d.db, d.tgClient, logger)
	if err != nil {
		return nil, fmt.Errorf("create driver: %w", err)
	}

	tg := afero.Fs(driver)
	if d.settings.Debug {
		tg = afero_tg.NewWrapper(tg, logger)
	}

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
