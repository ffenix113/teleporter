package template

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ffenix113/teleporter/manager"
)

//go:embed *.tpl
var embeddedTemplates embed.FS

func intQuery(query string, def int) (val int) {
	val = def

	if query != "" {
		val, _ = strconv.Atoi(query)
	}

	return val
}

type FileTree struct {
	Depth int
	Tree  manager.Tree
}

var tplFuncs = map[string]interface{}{
	"Limit": func(r *http.Request) int {
		return intQuery(r.URL.Query().Get("limit"), 100)
	},
	"Offset": func(r *http.Request) int {
		return intQuery(r.URL.Query().Get("offset"), 0)
	},
	"Mult": func(a, b int) int {
		return a * b
	},
	"MakeTree": func(depth int, tree manager.Tree) FileTree {
		return FileTree{
			Depth: depth + 1,
			Tree:  tree,
		}
	},
}

func ReadTemplates(templatesPath string) *template.Template {
	tpl := template.New("base").Funcs(tplFuncs)

	tplsFS := os.DirFS(templatesPath)
	if templatesPath == "" {
		tplsFS = embeddedTemplates
	}

	const tplSuffix = ".tpl"
	if err := fs.WalkDir(tplsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			panic(err)
		}

		if filepath.Ext(path) != tplSuffix {
			return nil
		}

		newTpl := tpl.New(strings.TrimSuffix(path, tplSuffix))

		tplData, _ := fs.ReadFile(tplsFS, path)

		if _, err := newTpl.Parse(string(tplData)); err != nil {
			panic(err)
		}

		return nil
	}); err != nil {
		panic(err)
	}

	return tpl
}
