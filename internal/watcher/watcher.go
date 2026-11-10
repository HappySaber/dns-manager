package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	path     string
	debounce time.Duration
	log      *slog.Logger
}

func New(path string, debounce time.Duration, log *slog.Logger) *Watcher {
	return &Watcher{path: path, debounce: debounce, log: log}
}

func (w *Watcher) Run(ctx context.Context, onChange func(context.Context)) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer fsw.Close()

	dir := filepath.Dir(w.path)
	if err := fsw.Add(dir); err != nil {
		return fmt.Errorf("watch %s: %w", dir, err)
	}
	name := filepath.Base(w.path)

	var (
		timer  *time.Timer
		timerC <-chan time.Time
	)
	for {
		select {
		case <-ctx.Done():
			return nil

		case ev, ok := <-fsw.Events:
			if !ok {
				return nil
			}
			if filepath.Base(ev.Name) != name || ev.Op == fsnotify.Chmod {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(w.debounce)
			} else {
				timer.Reset(w.debounce)
			}
			timerC = timer.C

		case <-timerC:
			timerC = nil
			onChange(ctx)

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			w.log.Warn("watcher error", "err", err)
		}
	}
}
