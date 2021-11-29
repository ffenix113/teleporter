package main

import (
	"context"
	"fmt"
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

	go web.Listen(":9000", cnf.App.TemplatePath, cl)

	cl.VerifyLocalFilesExist()

	// cl.TaskMonitor.Input <- arman92.NewUploadFile(cl, "data/test.txt")

	go func() {
		cwd, _ := os.Getwd()
		fsnotify.NewListener(path.Join(cwd, "data"), cl)
	}()

	select {}
}

func main2() {
	conf := config.Load()

	cl, _ := arman92.NewClient(context.Background(), conf)

	// rawUpdates gets all updates comming from tdlib
	rawUpdates := cl.Client.GetRawUpdatesChannel(100)
	for update := range rawUpdates {
		// Show all updates
		fmt.Println(update.Data)
		fmt.Print("\n\n")
	}
}
