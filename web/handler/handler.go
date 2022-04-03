package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/manager/arman92"
)

const MaxDownloadSize = 60 * 1024 * 1024 // 60 MB

type Handler struct {
	cl *arman92.Client
}

func NewHandler(cl *arman92.Client) *Handler {
	return &Handler{cl: cl}
}

func (h Handler) FileList(_ http.ResponseWriter, r *http.Request) ([]*manager.File, error) {
	pathKey := strings.TrimSuffix(chi.URLParam(r, "*"), "/")

	tree, ok := manager.FindInTree[*manager.Tree](h.cl.FileTree, pathKey)
	if !ok {
		return nil, ErrNotFound
	}

	return tree.FilesInfo(), nil
}

func (h Handler) PathDelete(_ http.ResponseWriter, r *http.Request) (NoResponse, error) {
	pathKey := strings.TrimSuffix(chi.URLParam(r, "*"), "/")

	if _, ok := manager.FindInTree[*manager.Tree](h.cl.FileTree, pathKey); !ok {
		return nil, ErrNotFound
	}

	if err := h.cl.DeletePath(r.Context(), pathKey); err != nil {
		return nil, fmt.Errorf("delete file with path %q error: %w", pathKey, err)
	}

	return nil, nil
}

func (h Handler) FileDownload(w http.ResponseWriter, r *http.Request) {
	pathKey := strings.TrimSuffix(chi.URLParam(r, "*"), "/")

	cachedFile, ok := manager.FindInTree[*manager.File](h.cl.FileTree, pathKey)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if cachedFile.Size > MaxDownloadSize {
		log.Printf("file is larger then limit: %d > %d", cachedFile.Size, MaxDownloadSize)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(pathKey))
	w.Header().Set("Content-Type", "application/octet-stream")

	dFile, err := os.Open(h.cl.AbsPath(pathKey))
	if err != nil {
		log.Printf("open file: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer dFile.Close()

	if _, err := io.Copy(w, dFile); err != nil {
		log.Printf("copy file on download: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
