package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/manager/arman92"
	"github.com/ffenix113/teleporter/tasks"
)

const MaxDownloadSize = 60 * 1024 * 1024 // 60 MB
const MaxUploadSize = 10 * 1024 * 1024   // 10 MB

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

	v, ok := manager.FindInTree[*manager.Tree](h.cl.FileTree, pathKey)
	if !ok {
		return nil, ErrNotFound
	}
	_ = v

	if err := h.cl.DeleteFile(r.Context(), pathKey); err != nil {
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

func (h Handler) FileUpload(w http.ResponseWriter, r *http.Request) (NoResponse, error) {
	pathKey := strings.TrimSuffix(chi.URLParam(r, "*"), "/")

	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		return nil, fmt.Errorf("parse multipart form error: %w", err)
	}

	ff, header, err := r.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("get file from form error: %w", err)
	}

	if header.Size > MaxUploadSize {
		return nil, fmt.Errorf("file is larger then limit: %d > %d", header.Size, MaxUploadSize)
	}

	f, err := os.CreateTemp(h.cl.TempPath, "*_"+header.Filename)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	defer os.RemoveAll(f.Name())

	if n, err := io.Copy(f, ff); err != nil && !errors.Is(err, io.EOF) {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("file is larger then limit: %d(or more) > %d", n, MaxUploadSize)
		}

		return nil, fmt.Errorf("copy file: %w", err)
	}

	if err := os.Rename(f.Name(), h.cl.AbsPath(path.Join(pathKey, header.Filename))); err != nil {
		return nil, fmt.Errorf("move uploaded file to data dir: %w", err)
	}

	c, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Wait for file upload and wait for its finish
	h.cl.AddPreAddHook(func(task tasks.Task) (tasks.Task, bool, error) {
		filePath := path.Join(pathKey, header.Filename)

		uploadTask, isUpload := task.(*arman92.UploadFile)
		if !isUpload || uploadTask.RelativePath != filePath {
			return task, false, nil
		}

		return arman92.WithCallback(task, func(task tasks.Task) {
			cancel()
		}), true, nil
	})
	<-c.Done()

	return nil, nil
}
