package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/fsnotify"
	"github.com/ffenix113/teleporter/manager/arman92"
	"github.com/ffenix113/teleporter/web"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	cnf := config.Load()

	cl, err := arman92.NewClient(context.Background(), cnf)
	if err != nil {
		panic(err)
	}

	go web.Listen(cnf.App.WebListen, cnf.App.TemplatePath, cl)

	cl.SynchronizeFiles()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	cwd, _ := os.Getwd()
	listener := fsnotify.NewListener(path.Join(cwd, "data"), cl)

	<-ctx.Done()
	listener.Close()
	log.Println("Shutdown", ctx.Err().Error())
}
