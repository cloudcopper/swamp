package main

import (
	"log/slog"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher      *fsnotify.Watcher
	log          *Logger
	closeWg      sync.WaitGroup
	ChanModified chan string
	ChanRemoved  chan string
}

func NewWatcher(log *Logger) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:      watcher,
		log:          log,
		ChanModified: make(chan string, 1),
		ChanRemoved:  make(chan string, 1),
	}

	w.closeWg.Add(1)
	go func() {
		defer w.closeWg.Done()
		defer close(w.ChanModified)
		defer close(w.ChanRemoved)
		w.process()
	}()

	return w, nil
}

func (w *Watcher) Close() {
	if w == nil {
		return
	}
	if w.watcher == nil {
		return
	}
	w.watcher.Close()
	w.closeWg.Wait()
	w.watcher = nil
}

func (w *Watcher) AddDir(path string) error {
	log := w.log
	log.Info("Watcher add dir", slog.String("path", path))
	err := w.watcher.Add(path)
	if err != nil {
		log.Error("Watcher add dir failed!!!", slog.Any("err", err), slog.String("path", path))
	}
	return err
}

func (w *Watcher) process() {
	log := w.log
	log.Info("Watcher process started")
	defer log.Warn("Watcher process complete!!!")

	for {
		select {
		case err, ok := <-w.watcher.Errors:
			if err != nil {
				log.Error("watcher error", slog.Any("err", err))
			}
			if !ok {
				return
			}
		case event, ok := <-w.watcher.Events:
			log.Debug("watcher event", slog.Any("event", event))
			if !ok {
				return
			}

			file := event.Name
			if event.Has(fsnotify.Create) {
				size := fileSize(file)
				log.Debug("file created", slog.String("file", file), slog.Int64("size", size))
				w.ChanModified <- file
			}
			if event.Has(fsnotify.Write) {
				size := fileSize(file)
				log.Debug("file modified", slog.String("file", file), slog.Int64("size", size))
				w.ChanModified <- file
			}
			if event.Has(fsnotify.Rename) {
				log.Debug("file renamed", slog.String("file", file))
				w.ChanRemoved <- file
			}
			if event.Has(fsnotify.Remove) {
				log.Debug("file removed", slog.String("file", file))
				w.ChanRemoved <- file
			}
		}
	}
}
