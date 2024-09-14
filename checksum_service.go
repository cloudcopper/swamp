package swamp

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

type ChecksumService struct {
	log             *ports.Logger
	watcher         InputWatcherService
	artifactStorage ports.ArtifactStorage
	repositories    domain.Repositories
	closeWg         sync.WaitGroup
}

func NewChecksumService(log *ports.Logger, watcher InputWatcherService, artifactStorage ports.ArtifactStorage, repositories domain.Repositories) (*ChecksumService, error) {
	log = log.With(slog.String("entity", "ChecksumService"))

	if _, err := repositories.Repo().FindAll(); err != nil {
		return nil, err
	}

	s := &ChecksumService{
		log:             log,
		watcher:         watcher,
		artifactStorage: artifactStorage,
		repositories:    repositories,
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
	artifactStorage := s.artifactStorage

	repos, err := s.repositories.Repo().FindAll()
	if err != nil {
		log.Error("unable to read all repos", slog.Any("err", err))
		return
	}

	for path := range modified {
		log := log.With(slog.String("path", path))
		log.Debug("detect modified")

		for _, repo := range repos {
			// Check the path belongs to repo
			if !strings.HasPrefix(path, repo.Input) {
				continue
			}
			log := log.With(slog.String("repo", repo.Name))
			log.Debug("path match repo")

			// Check the path is a good checksum
			goodFiles, badFiles, err := adapters.CheckChecksum(log, path)
			if err == errors.ErrIsNotChecksumFile {
				continue
			}
			if err != nil {
				log.Error("unable to verify checksum file", slog.Any("goodFiles", goodFiles), slog.Any("badFile", badFiles), slog.Any("err", err))
				break
			}
			log.Info("checksum file verified", slog.Any("goodFiles", goodFiles))

			// Create new artifacts
			artifacts := append(goodFiles, path)
			id := lib.GetFirstSubdir(repo.Input, path)
			artifactId, createdAt, err := artifactStorage.NewArtifact(repo, artifacts, models.ArtifactID(id))
			if err != nil {
				log.Error("unable to create new artifacts", slog.Any("err", err))
			}
			log.Info("new artifact created", slog.String("artifactId", string(artifactId)))

			// Cleanup input artifacts
			log.Info("cleanup input artifacts")
			input := repo.Input
			for i := range artifacts {
				// clean up in reverse order, so the checksum file is removed first
				file := artifacts[len(artifacts)-i-1]
				lib.Assert(strings.HasPrefix(file, input))
				dir := lib.GetFirstSubdir(input, file)
				name := filepath.Join(input, dir)
				if dir == "" {
					lib.Assert(filepath.IsAbs(file))
					name = file
				} else {
					if !lib.IsDirectoryExist(name) {
						continue
					}
				}
				if err := os.RemoveAll(name); err != nil {
					log.Warn("unable to remove input artifact", slog.Any("err", err), slog.String("name", name))
				}
			}

			// Insert artifact record
			artifact := &models.Artifact{
				ID:        artifactId,
				RepoName:  repo.Name,
				CreatedAt: createdAt,
			}
			if err := s.repositories.Artifact().Insert(artifact); err != nil {
				log.Error("unable insert artifact record", slog.String("artifactId", string(artifactId)), slog.Any("err", err))
			}

			break
		}
	}
}
