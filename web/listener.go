package web

import (
	"net/http"

	"github.com/ffenix113/teleporter/manager/arman92"
)

func Listen(listenAddr string, templatesPath string, cl *arman92.Client) {
	err := http.ListenAndServe(listenAddr, NewRouter(cl, templatesPath))

	panic(err)
}
