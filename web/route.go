package web

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/manager/arman92"
	"github.com/ffenix113/teleporter/web/template"
)

func NewRouter(cl *arman92.Client, templatesPath string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(CORS)

	r.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
		// pathKey := strings.TrimSuffix(chi.URLParam(r, "*"), "/")

		var filesAtPath []manager.File // cl.PinnedHeader.FilesAtPath(pathKey)

		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filesAtPath)
	})

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

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Access-Control-Allow-Origin", "*")

		next.ServeHTTP(writer, request)
	})
}