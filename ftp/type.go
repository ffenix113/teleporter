package ftp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math/big"
	"net"
	"time"

	"github.com/Arman92/go-tdlib/v2/client"
	"github.com/Arman92/go-tdlib/v2/tdlib"
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

	tlsConf *tls.Config

	logger *zap.Logger
}

func NewDriver(db *bun.DB, tgClient *arman92.Client, settings config.FTP, logger *zap.Logger) *Driver {
	storageOptimizer(tgClient.TDClient, logger, settings.Optimize)()

	tgClient.AddUpdateHandler(UpdateDeletedMessages(db, logger))

	return &Driver{
		db:       db,
		tgClient: tgClient,
		settings: settings,
		logger:   logger,
		tlsConf:  loadCerts(), // generateTLSConfig(),
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

	logger := d.logger.With(zap.String("user", user))

	driver, err := afero_tg.NewTelegram(cc, user, pass, d.db, d.tgClient, logger)
	if err != nil {
		return nil, fmt.Errorf("create driver: %w", err)
	}

	tg := afero.Fs(driver)
	if d.settings.Debug {
		tg = afero_tg.NewWrapper(tg, logger)
	}

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
	return d.tlsConf, nil
}

func loadCerts() *tls.Config {
	cert, err := tls.LoadX509KeyPair("./cert.pem", "./key.pem")
	if err != nil {
		log.Fatal(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}

func generateTLSConfig() *tls.Config {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"server"},
		},
		DNSNames:  []string{"localhost"},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemCert == nil {
		log.Fatal("Failed to encode certificate to PEM")
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if pemKey == nil {
		log.Fatal("Failed to encode key to PEM")
	}

	cert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	return &tls.Config{Certificates: []tls.Certificate{cert}}
}

func UpdateDeletedMessages(db *bun.DB, logger *zap.Logger) arman92.UpdateHandler {
	return func(update tdlib.UpdateMsg) bool {
		if update.Data["@type"].(string) != string(tdlib.ChatEventMessageDeletedType) {
			return false
		}

		var upd tdlib.ChatEventMessageDeleted
		json.Unmarshal(update.Raw, &upd)

		exists, err := db.NewSelect().Model((*afero_tg.User)(nil)).
			Where("chat_id = ?", upd.Message.ChatID).
			Exists(context.Background())
		if err != nil {
			logger.Error("select user on deleted message", zap.Error(err))
			return false
		}

		if !exists {
			return false
		}

		res, err := db.NewDelete().Model((*afero_tg.DBFileInfo)(nil)).
			Where("chat_id = ? AND message_id = ?", upd.Message.ChatID, upd.Message.ID).
			Exec(context.Background())
		if err != nil {
			logger.Error("delete file info on deleted message", zap.Error(err))
			return false
		}

		affected, err := res.RowsAffected()
		if err != nil {
			logger.Error("get affected rows on deleted message", zap.Error(err))
			return false
		}

		if affected == 0 {
			logger.Warn("no rows affected on deleted message", zap.Int64("chat_id", upd.Message.ChatID), zap.Int64("message_id", upd.Message.ID))
		}

		return false
	}
}

func storageOptimizer(cl *client.Client, logger *zap.Logger, conf *config.Optimize) func() {
	ticker := time.NewTicker(conf.Interval)
	forceChan := make(chan struct{}, 1)

	forceOptimize := func() {
		select {
		case forceChan <- struct{}{}:
		default:
		}
	}

	go func() {
		for {
			select {
			case <-ticker.C:
			case <-forceChan:
			}

			deletedStats, err := cl.OptimizeStorage(conf.MaxTotalSize, int32(conf.UnaccessedDuration.Seconds()), conf.MaxFilesCount, int32(conf.Immunity.Seconds()), nil, nil, nil, true, 5)
			if err != nil {
				logger.Error("optimize storage", zap.Error(err))
			}

			if deletedStats != nil && deletedStats.Count != 0 {
				logger.Info("optimized storage", zap.String("size", sizeStringify(deletedStats.Size)), zap.Int32("count", deletedStats.Count))
			}

			ticker.Reset(conf.Interval)
		}
	}()

	return forceOptimize
}

func sizeStringify(size int64) string {
	sizes := []string{"B", "kB", "MB", "GB"}

	var i int
	for size >= 1024 {
		size /= 1024
		i++
	}

	return fmt.Sprintf("%d%s", size, sizes[i])
}
