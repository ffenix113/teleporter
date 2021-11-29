package template

import (
	"html/template"
	"io/fs"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func intQuery(query string, def int) (val int) {
	val = def

	if query != "" {
		val, _ = strconv.Atoi(query)
	}

	return val
}

var tplFuncs = map[string]interface{}{
	"Limit": func(r *http.Request) int {
		return intQuery(r.URL.Query().Get("limit"), 100)
	},
	"Offset": func(r *http.Request) int {
		return intQuery(r.URL.Query().Get("offset"), 0)
	},
}

func ReadTemplates(templatesPath string) *template.Template {
	tpl := template.New("base").Funcs(tplFuncs)

	if err := filepath.WalkDir(templatesPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			panic(err)
		}

		if filepath.Ext(path) != ".tpl" {
			return nil
		}

		newTpl := tpl.New(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))

		tplData, _ := ioutil.ReadFile(path)

		if _, err := newTpl.Parse(string(tplData)); err != nil {
			panic(err)
		}

		return nil
	}); err != nil {
		panic(err)
	}

	return tpl
}
