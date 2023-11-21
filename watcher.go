package main

import "github.com/fsnotify/fsnotify"

type watcher struct {
	w *fsnotify.Watcher
}

func newWatcher() (*watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &watcher{
		w: w,
	}, nil
}

func (w *watcher) Close() {
	if w == nil {
		return
	}
	if w.w == nil {
		return
	}
	w.w.Close()
	w.w = nil
}
