package fsnotify

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/ffenix113/teleporter/manager/arman92"
	"github.com/ffenix113/teleporter/tasks"
)

const MaxFileSize = 350 * 1024 // 350kb

func NewListener(path string, cl *arman92.Client) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	processFunc := NewProcessEventFunc(cl, watcher)

	debouncer := NewDebounce(time.Second)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)

				// Do not store cache and temp files.
				if strings.HasSuffix(event.Name, "~") {
					continue
				}

				debouncer.Add(event, processFunc)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = AddRecursively(watcher, path)
	if err != nil {
		log.Fatal(err)
	}
}

type ProcessEventFunc func(event fsnotify.Event)

func NewProcessEventFunc(cl *arman92.Client, watcher *fsnotify.Watcher) func(fsnotify.Event) {
	return func(event fsnotify.Event) {
		log.Printf("op: %s, file: %q\n", event.Op, event.Name)

		switch {
		case IsOp(event.Op, fsnotify.Create):
			stat, err := os.Stat(event.Name)
			if err != nil {
				log.Printf("stat new item: %s", err)
				return
			}

			if stat.IsDir() {
				if err := watcher.Add(event.Name); err != nil {
					log.Printf("add new dir: %s", err.Error())
				}
				return
			}
			if fileSize := stat.Size(); fileSize != 0 {
				if kbs := fileSize / 1024; kbs > MaxFileSize {
					cl.AddTask(arman92.NewStaticTask(event.Name, arman92.NewCommon(nil, "UploadFile", tasks.TaskStatusError, fmt.Sprintf("file is larger than supported: %d > %d KB", kbs, MaxFileSize))))
					return
				}
				cl.AddTask(arman92.NewUploadFile(cl, event.Name))
			}
		case IsOp(event.Op, fsnotify.Write):
			stat, err := os.Stat(event.Name)
			if err != nil {
				log.Printf("stat new item: %s", err)
				return
			}
			if fileSize := stat.Size(); fileSize != 0 {
				if kbs := fileSize / 1024; kbs > MaxFileSize {
					cl.AddTask(arman92.NewStaticTask(event.Name, arman92.NewCommon(nil, "UploadFile", tasks.TaskStatusError, fmt.Sprintf("file is larger than supported: %d > %d KB", kbs, MaxFileSize))))
					return
				}
				cl.AddTask(arman92.NewUploadFile(cl, event.Name))
				return
			}
			cl.AddTask(arman92.NewUploadFile(cl, event.Name))
		case IsOp(event.Op, fsnotify.Remove) || IsOp(event.Op, fsnotify.Rename):
			cl.AddTask(arman92.NewDeleteFile(cl, event.Name))
		}
	}
}

func AddRecursively(w *fsnotify.Watcher, dirPath string) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			if err := w.Add(path); err != nil {
				return err
			}
		}

		return err
	})
}

func IsOp(orig, compareTo fsnotify.Op) bool {
	return orig&compareTo == compareTo
}
