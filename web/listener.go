package web

import (
	"log"
	"net/http"

	"github.com/ffenix113/teleporter/manager/arman92"
	"github.com/ffenix113/teleporter/web/template"
)

func Listen(listenAddr string, templatesPath string, cl *arman92.Client) {
	err := http.ListenAndServe(listenAddr, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("new request: %s %s", request.Method, request.URL.String())
		tpls := template.ReadTemplates(templatesPath)
		tplName := request.RequestURI[1:]

		if tplName == "" {
			tplName = "index"
		}

		tpl := tpls.Lookup(tplName)
		if tpl == nil {
			http.NotFound(writer, request)
			return
		}

		if err := tpl.Execute(writer, map[string]interface{}{
			"request": request,
			"client":  cl,
		}); err != nil {
			panic(err)
		}
	}))

	panic(err)
}
