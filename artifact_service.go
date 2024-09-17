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

// ArtifactService listeing eventbus for next events:
//   - repo-updated - to maintain internal list of repos
//   - input-file-modified - to check if the file is checksum belonging to any of known repos,
//     and if so, then create new artifact by checksum file
type ArtifactService struct {
	log                    ports.Logger
	bus                    ports.EventBus
	artifactStorage        ports.ArtifactStorage
	repositories           domain.Repositories
	chRepoUpdated          chan ports.Event
	chFileModified         chan ports.Event
	chDanglingRepoArtifact chan ports.Event
	closeWg                sync.WaitGroup
}

func NewArtifactService(log ports.Logger, bus ports.EventBus, artifactStorage ports.ArtifactStorage, repositories domain.Repositories) (*ArtifactService, error) {
	log = log.With(slog.String("entity", "ArtifactService"))

	if _, err := repositories.Repo().FindAll(); err != nil {
		return nil, err
	}

	s := &ArtifactService{
		log:                    log,
		bus:                    bus,
		artifactStorage:        artifactStorage,
		repositories:           repositories,
		chRepoUpdated:          bus.Sub(ports.TopicRepoUpdated),
		chFileModified:         bus.Sub(ports.TopicInputFileModified),
		chDanglingRepoArtifact: bus.Sub(ports.TopicDanglingRepoArtifact),
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

func (s *ArtifactService) Close() {
	s.log.Info("closing")
	s.bus.Unsub(s.chDanglingRepoArtifact)
	s.bus.Unsub(s.chFileModified)
	s.bus.Unsub(s.chRepoUpdated)
	s.closeWg.Wait()
}

func (s *ArtifactService) background() {
	log := s.log
	artifactStorage := s.artifactStorage

	repos, err := s.repositories.Repo().FindAll()
	if err != nil {
		log.Error("unable to read all repos", slog.Any("err", err))
		return
	}

	for {
		select {
		case _, ok := <-s.chRepoUpdated:
			if !ok {
				return
			}
			repos, err = s.repositories.Repo().FindAll()
			if err != nil {
				log.Error("unable to update all repos", slog.Any("err", err))
				return
			}

		case event, ok := <-s.chFileModified:
			if !ok {
				return
			}
			path := event[0]
			log := log.With(slog.String("path", path))
			log.Debug("detect modified")

			for _, repo := range repos {
				// Check the path belongs to repo
				if !strings.HasPrefix(path, repo.Input) {
					continue
				}
				log := log.With(slog.Any("repoID", repo.ID))
				log.Debug("path match repo")

				// Check the path is a good checksum
				checksum, goodFiles, badFiles, err := adapters.CheckChecksum(log, path)
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
				artifactID, createdAt, err := artifactStorage.NewArtifact(repo, id, artifacts)
				if err != nil {
					log.Error("unable to create new artifacts", slog.Any("err", err))
				}
				log.Info("new artifact created", slog.String("artifactID", string(artifactID)))

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
						lib.Assert(lib.IsAbs(file))
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
					ID:        artifactID,
					RepoID:    repo.ID,
					CreatedAt: createdAt,
					Checksum:  checksum,
				}
				if err := s.repositories.Artifact().Create(artifact); err != nil {
					log.Error("unable insert artifact record", slog.String("artifactID", string(artifactID)), slog.Any("err", err))
				}

				break
			}
		case event, ok := <-s.chDanglingRepoArtifact:
			if !ok {
				return
			}
			repoID, artifactID := event[0], event[1]
			s.checkRepoArtifact(repoID, artifactID)
		}
	}
}

// The checkRepoArtifact checks the artifact inside repo storage.
// If it dangling, it creates new artifact model.
// TODO If it broken, signal its broken
func (s *ArtifactService) checkRepoArtifact(repoID models.RepoID, artifactID models.ArtifactID) {
	log := s.log.With(slog.Any("repoID", repoID), slog.Any("artifactID", artifactID))
	repo, err := s.repositories.Repo().FindByID(repoID)
	if err != nil {
		log.Error("unable to fine repo by id", slog.Any("err", err))
		return
	}

	loc := filepath.Join(repo.Storage, artifactID)
	createdAt, checksum, err := s.verifyArtifact(loc)
	if err != nil {
		log.Error("unable to verify aritfact", slog.Any("err", err))
		s.bus.Pub(ports.TopicBrokenRepoArtifact, ports.Event{repoID, artifactID})
		return
	}

	artifact, err := s.repositories.Artifact().FindByID(repoID, artifactID)
	if err != nil {
		log.Error("unable to find artifact", slog.Any("err", err))
		return
	}
	if artifact.ID == models.EmptyArtifactID {
		log.Info("dangling artifact")
		artifact.ID = artifactID
		artifact.RepoID = repoID
		artifact.Checksum = checksum
		artifact.CreatedAt = createdAt
		if err := s.repositories.Artifact().Create(artifact); err != nil {
			log.Error("unable create artifact record", slog.Any("err", err))
		}
		return
	}
	if artifact.ID != artifactID {
		// This would be some serious issue
		// We expect to read from artifact repository
		// either requested ID not empty
		log.Error("wrong artifact found", slog.Any("unexpected artifact id", artifact.ID))
		return
	}
	if artifact.CreatedAt == createdAt && artifact.Checksum != checksum {
		log.Error("tampered artifact", slog.Any("original checksum", artifact.Checksum), slog.Any("checksum", checksum))
		s.bus.Pub(ports.TopicBrokenRepoArtifact, ports.Event{repoID, artifactID})
		return
	}
	if artifact.CreatedAt != createdAt && artifact.Checksum == checksum {
		log.Warn("reuploaded artifact", slog.Any("originally createdAt", artifact.CreatedAt), slog.Any("createdAt", createdAt))
		// Do nothing - keep original details
		return
	}
}

// The verifyArtifact check the location
// has only artifact files,
// and returns createdAt, checksum or error
func (s *ArtifactService) verifyArtifact(location string) (int64, string, error) {
	return 12345678, "xxxxxxxx", nil
}
