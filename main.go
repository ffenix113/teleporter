package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/ftp"
	"github.com/ffenix113/teleporter/manager/arman92"

	adapter "github.com/fclairamb/go-log/zap"
	"go.uber.org/zap"
)

func main() {
	logCnf := zap.NewDevelopmentConfig()
	logCnf.Level.SetLevel(zap.DebugLevel)
	logger, _ := logCnf.Build(zap.WithCaller(true))

	logger.Debug("loading config")
	cnf := config.Load()

	logger.Debug("create client")
	cl, err := arman92.NewClient(context.Background(), cnf)
	if err != nil {
		panic(err)
	}

	dbConn, err := sql.Open("pgx", cnf.DB.DSN)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	if err := dbConn.PingContext(ctx); err != nil {
		panic(err)
	}
	cancel()
	logger.Debug("db available")

	db := bun.NewDB(dbConn, pgdialect.New())

	ftpServer := ftpserver.NewFtpServer(ftp.NewDriver(db, cl, cnf.FTP, logger))
	ftpServer.Logger = adapter.NewWrap(logger.Sugar())

	go func() {
		log.Printf("start ftp server on addr %q\n", cnf.FTP.ListenAddr)
		if err := ftpServer.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	logger.Debug("waiting for exit")
	<-ctx.Done()
	ftpServer.Stop()
	logger.Debug("Shutdown", zap.Error(ctx.Err()))
}
