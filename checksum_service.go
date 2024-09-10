package main

import (
	"log/slog"
	"sync"

	"xorm.io/xorm"
)

type ChecksumService struct {
	log     *Logger
	watcher InputWatcherService
	engine  *xorm.Engine
	closeWg sync.WaitGroup
}

func NewChecksumService(log *Logger, watcher InputWatcherService, engine *xorm.Engine) (*ChecksumService, error) {
	log = log.With(slog.String("entity", "ChecksumService"))

	if _, err := FindAll[Repo](engine); err != nil {
		return nil, err
	}

	s := &ChecksumService{
		log:     log,
		watcher: watcher,
		engine:  engine,
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

func (s *ChecksumService) Close() {
	if s == nil {
		return
	}
	if s.watcher == nil {
		return
	}

	s.log.Info("closing")
	s.closeWg.Wait()
	s.watcher = nil
}

func (s *ChecksumService) background() {
	log := s.log
	modified := s.watcher.GetChanModified()

	repos, err := FindAll[Repo](s.engine)
	if err != nil {
		log.Error("unable to read all repos", slog.Any("err", err))
		return
	}

	for path := range modified {
		log := log.With(slog.String("path", path))
		log.Debug("detect modified")

		for _, repo := range repos {
			// Check the path belongs to repo
			if !repo.IsPathInInput(path) {
				continue
			}
			log := log.With(slog.String("repo", repo.Name))
			log.Debug("path match repo")

			// Check the path is a good checksum
			goodFiles, badFiles, err := CheckChecksum(log, path)
			if err == ErrIsNotChecksumFile {
				continue
			}
			if err != nil {
				log.Error("unable to verify checksum file", slog.Any("goodFiles", goodFiles), slog.Any("badFile", badFiles), slog.Any("err", err))
				break
			}
			log.Info("checksum file verified", slog.Any("goodFiles", goodFiles))

			// TODO Move artifact to storage
		}
	}
}
