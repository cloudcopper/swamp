package swamp

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

type RepoService struct {
	log           ports.Logger
	bus           ports.EventBus
	walk          infra.FilepathWalk
	repositories  domain.Repositories
	chRepoUpdated chan ports.Event
	closeWg       sync.WaitGroup
}

func NewRepoService(log ports.Logger, bus ports.EventBus, walk infra.FilepathWalk, repositories domain.Repositories) *RepoService {
	log = log.With(slog.String("entity", "RepoService"))
	s := &RepoService{
		log:           log,
		bus:           bus,
		walk:          walk,
		repositories:  repositories,
		chRepoUpdated: bus.Sub(ports.TopicRepoUpdated),
	}

	s.closeWg.Add(1)
	go func() {
		defer s.closeWg.Done()
		log.Info("process started")
		defer log.Warn("process complete")
		s.background()
	}()

	return s
}

func (s *RepoService) Close() {
	s.log.Info("closing")
	s.bus.Unsub(s.chRepoUpdated)
	s.closeWg.Wait()
}

func (s *RepoService) background() {
	for ids := range s.chRepoUpdated {
		for _, id := range ids {
			s.checkRepoById(id)
		}
	}
}

func (s *RepoService) checkRepoById(repoID models.RepoID) {
	repo, err := s.repositories.Repo().FindByID(repoID)
	if err != nil {
		s.log.Error("unable to find repo", slog.Any("repoID", repoID), slog.Any("err", err))
		return
	}
	s.checkRepo(repo)
}

func (s *RepoService) checkRepo(repo *models.Repo) {
	log := s.log.With(slog.Any("repoID", repo.ID))
	log.Debug("check repo")

	// TODO Abstract out storage!!!!
	storage := repo.Storage
	if !lib.IsDirectoryExist(storage) {
		log.Error("storage not found", slog.String("storage", storage))
		return
	}

	//
	// Check repo's artifacts
	//
	s.walk.Walk(storage, func(name string, err error) (bool, error) {
		if err != nil {
			log.Error("walk error", slog.String("name", name), slog.Any("err", err))
			return true, nil
		}
		if !adapters.IsChecksumFile(name) {
			return true, nil
		}

		// the name is checksum file within repo's storage
		artifactID := filepath.Base(filepath.Dir(name))
		artifact, err := s.repositories.Artifact().FindByID(repo.ID, artifactID)
		if err != nil {
			log := log.With(slog.Any("repoID", repo.ID), slog.Any("artifactID", artifactID))
			log.Error("unable to find artifact", slog.Any("err", err))
			return true, nil
		}
		if artifact.ID == artifactID {
			return true, nil
		}

		// This is dangling artifact
		// It presents in repo storage but not in database
		// There might be few reasons for that:
		// - we just starting up
		// - it was manually written to storage
		// - it was written by other instance or means
		s.bus.Pub(ports.TopicDanglingRepoArtifact, ports.Event{repo.ID, artifactID})
		return true, nil
	})
}
