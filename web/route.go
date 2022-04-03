package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/manager/arman92"
	"github.com/ffenix113/teleporter/web/handler"
	"github.com/ffenix113/teleporter/web/template"
)

type Middleware func(http.Handler) http.Handler

func NewRouter(conf config.Config, cl *arman92.Client, templatesPath string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer,
		middleware.RealIP,
		middleware.Logger,
		middleware.Compress(6),
		CORS,
		middleware.RedirectSlashes,
		middleware.CleanPath,
		IPWhitelist(conf.App.IPWhitelist),
	)

	h := handler.NewHandler(cl)

	r.Get("/files/list", handler.Wrap(h.FileList)) // Route to match '/files/list/'
	r.Get("/files/list/*", handler.Wrap(h.FileList))
	r.Get("/files/download/*", h.FileDownload)
	r.Delete("/files/delete/*", handler.Wrap(h.PathDelete))
	// This is route to show tasks.
	// Better would be to use Vue instead.
	r.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		tpls := template.ReadTemplates(templatesPath)
		tplName := request.RequestURI[1:]

		if tplName == "" {
			tplName = "index.html"
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
	})

	return r
}
