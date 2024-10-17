package infra

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
)

type WatcherService struct {
	id                  string
	log                 ports.Logger
	bus                 ports.EventBus
	chTopicInputUpdated chan ports.Event
	watcher             *fsnotify.Watcher
	closeWg             sync.WaitGroup
}

func NewWatcherService(id string, log ports.Logger, bus ports.EventBus) (*WatcherService, error) {
	log = log.With(slog.String("entity", "WatcherService"), slog.String("id", id))
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	s := &WatcherService{
		id:                  id,
		log:                 log,
		bus:                 bus,
		chTopicInputUpdated: bus.Sub(ports.TopicInputUpdated),
		watcher:             watcher,
	}
	log.Info("created")

	s.closeWg.Add(1)
	go func() {
		defer s.closeWg.Done()
		log.Info("process started")
		defer log.Warn("process complete")
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
	s.bus.Unsub(s.chTopicInputUpdated)
	s.watcher.Close()
	s.closeWg.Wait()
	s.watcher = nil
}

// WARN The remove of path would remove if from watch list inside the fsnotify!!!
// WARN	Such even would be communcated by remove event with name of path
// TODO Handle reassignment transparently in process(). Make the test
// TODO Auto assing recursive directiry creation. Do not watch once removed. Make the test
func (s *WatcherService) addDir(path string) error {
	log := s.log
	lib.Assert(lib.IsAbs(path))
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
	log, bus, fs := s.log, s.bus, afero.NewOsFs()
	topicFileModified := fmt.Sprintf("%v-file-modified", s.id)
	topicFileRemoved := fmt.Sprintf("%v-file-removed", s.id)
	for {
		select {
		case event, ok := <-s.chTopicInputUpdated:
			log.Debug("watcher event", slog.Any("event", event))
			if !ok {
				return
			}
			for _, path := range event {
				s.addDir(path)
			}
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
			exist, _ := afero.DirExists(fs, file)
			if event.Has(fsnotify.Create) && exist {
				dir := file
				log := log.With(slog.String("dir", dir))
				log.Debug("directory created")
				err := s.addDir(dir)
				if err != nil {
					log.Error("unable to add recursive dir")
				}
				continue
			}
			if event.Has(fsnotify.Create) {
				size := lib.FileSize(fs, file)
				log.Debug("file created", slog.String("file", file), slog.Int64("size", size))
				bus.Pub(topicFileModified, ports.Event{file})
			}
			if event.Has(fsnotify.Write) {
				size := lib.FileSize(fs, file)
				log.Debug("file modified", slog.String("file", file), slog.Int64("size", size))
				bus.Pub(topicFileModified, ports.Event{file})
			}
			if event.Has(fsnotify.Rename) {
				log.Debug("file renamed", slog.String("file", file))
				bus.Pub(topicFileRemoved, ports.Event{file})
			}
			if event.Has(fsnotify.Remove) {
				log.Debug("file removed", slog.String("file", file))
				bus.Pub(topicFileRemoved, ports.Event{file})
			}
		}
	}
}
