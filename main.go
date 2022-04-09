package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/fsnotify"
	"github.com/ffenix113/teleporter/manager/arman92"
	"github.com/ffenix113/teleporter/web"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Llongfile | log.Lmicroseconds)
	log.Println("hello")

	cnf := config.Load()

	log.Println("create client")
	cl, err := arman92.NewClient(context.Background(), cnf)
	if err != nil {
		panic(err)
	}

	log.Println("starting web server")
	go web.Listen(cnf, cnf.App.WebListen, cnf.App.TemplatePath, cl)

	log.Println("update files state on start")
	if err := cl.SynchronizeFiles(); err != nil {
		panic(err)
	}
	log.Println("update files state on start done")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	log.Println("start file listener")
	listener := fsnotify.NewListener(cnf.App.FilesPath, cl)

	log.Println("waiting for exit")
	<-ctx.Done()
	listener.Close()
	log.Println("Shutdown", ctx.Err().Error())
}
