package swamp

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"github.com/fsnotify/fsnotify"
)

type WatcherService struct {
	id           string
	log          *ports.Logger
	watcher      *fsnotify.Watcher
	chanModified chan string
	chanRemoved  chan string
	closeWg      sync.WaitGroup
}

type InputWatcherService interface {
	GetChanModified() chan string
}

func (s *WatcherService) GetChanModified() chan string {
	return s.chanModified
}

func (s *WatcherService) GetChanRemoved() chan string {
	return s.chanRemoved
}

func NewWatcherService(log *ports.Logger, id string) (*WatcherService, error) {
	log = log.With(slog.String("entity", "WatcherService"), slog.String("id", id))
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	s := &WatcherService{
		id:           id,
		log:          log,
		watcher:      watcher,
		chanModified: make(chan string, 1),
		chanRemoved:  make(chan string, 1),
	}
	log.Info("created")

	s.closeWg.Add(1)
	go func() {
		defer s.closeWg.Done()
		log.Info("process started")
		defer log.Warn("process complete")
		defer close(s.chanModified)
		defer close(s.chanRemoved)
		s.background()
	}()

	return s, nil
}

func (s *WatcherService) Close() {
	if s == nil {
		return
	}
	if s.watcher == nil {
		return
	}

	s.log.Info("closing")
	s.watcher.Close()
	s.closeWg.Wait()
	s.watcher = nil
}

// WARN The remove of path would remove if from watch list
// WARN	Such even would be communcated by remove event with name of path
// TODO Handle reassignment transparently in process(). Make the test
// TODO Auto assing recursive directiry creation. Do not watch once removed. Make the test
func (s *WatcherService) AddDir(path string) error {
	log := s.log
	if abspath, err := filepath.Abs(path); abspath != path || err != nil {
		log.Error("add dir failed!!!", slog.Any("err", err), slog.String("path", path), slog.String("abspath", abspath))
		return errors.ErrMustBeAbsPath
	}
	log.Info("add dir", slog.String("path", path))
	err := s.watcher.Add(path)
	if err != nil {
		log.Error("add dir failed!!!", slog.Any("err", err), slog.String("path", path))
	}
	return err
}

func (s *WatcherService) background() {
	log := s.log
	for {
		select {
		case err, ok := <-s.watcher.Errors:
			if err != nil {
				log.Error("watcher error", slog.Any("err", err))
			}
			if !ok {
				return
			}
		case event, ok := <-s.watcher.Events:
			log.Debug("watcher event", slog.Any("event", event))
			if !ok {
				return
			}

			file := event.Name
			if event.Has(fsnotify.Create) && lib.IsDirectoryExist(file) {
				dir := file
				log := log.With(slog.String("dir", dir))
				log.Debug("directory created")
				err := s.AddDir(dir)
				if err != nil {
					log.Error("unable to add recursive dir")
				}
				continue
			}
			if event.Has(fsnotify.Create) {
				size := lib.FileSize(file)
				log.Debug("file created", slog.String("file", file), slog.Int64("size", size))
				s.chanModified <- file
			}
			if event.Has(fsnotify.Write) {
				size := lib.FileSize(file)
				log.Debug("file modified", slog.String("file", file), slog.Int64("size", size))
				s.chanModified <- file
			}
			if event.Has(fsnotify.Rename) {
				log.Debug("file renamed", slog.String("file", file))
				s.chanRemoved <- file
			}
			if event.Has(fsnotify.Remove) {
				log.Debug("file removed", slog.String("file", file))
				s.chanRemoved <- file
			}
		}
	}
}
