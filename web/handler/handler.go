package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/manager/arman92"
)

type Handler struct {
	cl *arman92.Client
}

func NewHandler(cl *arman92.Client) *Handler {
	return &Handler{cl: cl}
}

func (h Handler) FileList(w http.ResponseWriter, r *http.Request) {
	pathKey := strings.TrimSuffix(chi.URLParam(r, "*"), "/")

	tree, ok := manager.FindInTree[*manager.Tree](h.cl.FileTree, pathKey)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(tree.FilesInfo())
}

func (h Handler) PathDelete(w http.ResponseWriter, r *http.Request) {
	pathKey := strings.TrimSuffix(chi.URLParam(r, "*"), "/")

	if _, ok := manager.FindInTree[*manager.Tree](h.cl.FileTree, pathKey); !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := h.cl.DeletePath(r.Context(), pathKey); err != nil {
		log.Printf("delete file with path %q error: %s", pathKey, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
