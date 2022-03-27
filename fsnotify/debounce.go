package fsnotify

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Debounce struct {
	debounceDuration time.Duration

	evs   map[string]item
	evsMu sync.Mutex
}

type item struct {
	op    fsnotify.Op
	timer *time.Timer
}

func NewDebounce(debounceDuration time.Duration) *Debounce {
	return &Debounce{
		debounceDuration: debounceDuration,
		evs:              map[string]item{},
	}
}

func (d *Debounce) Add(event fsnotify.Event, eventFunc ProcessEventFunc) {
	d.evsMu.Lock()
	defer d.evsMu.Unlock()
	// Make paths uniform Unix style
	event.Name = filepath.ToSlash(event.Name)

	existing, ok := d.evs[event.Name]
	if !ok {
		funcTimer := time.AfterFunc(d.debounceDuration, func() {
			d.evsMu.Lock()
			delete(d.evs, event.Name)
			d.evsMu.Unlock()

			eventFunc(event)
		})

		existing = item{
			op:    event.Op,
			timer: funcTimer,
		}

		d.evs[event.Name] = existing
	}

	existing.timer.Reset(d.debounceDuration)
}
