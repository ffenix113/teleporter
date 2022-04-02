package web

import (
	"net/http"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/manager/arman92"
)

func Listen(conf config.Config, listenAddr string, templatesPath string, cl *arman92.Client) {
	err := http.ListenAndServe(listenAddr, NewRouter(conf, cl, templatesPath))

	panic(err)
}
