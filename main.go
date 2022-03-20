package main

import (
	"context"
	"os"
	"path"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/fsnotify"
	"github.com/ffenix113/teleporter/manager/arman92"
	"github.com/ffenix113/teleporter/web"
)

func main() {
	cnf := config.Load()

	cl, err := arman92.NewClient(context.Background(), cnf)
	if err != nil {
		panic(err)
	}

	go web.Listen(cnf.App.WebListen, cnf.App.TemplatePath, cl)

	cl.SynchronizeFiles()

	go func() {
		cwd, _ := os.Getwd()
		fsnotify.NewListener(path.Join(cwd, "data"), cl)
	}()

	select {}
}
