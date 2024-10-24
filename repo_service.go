package swamp

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra/disk"
	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
)

type RepoService struct {
	log                ports.Logger
	bus                ports.EventBus
	walk               disk.FilepathWalk
	repoRepository     domain.RepoRepository
	chTopicRepoUpdated chan ports.Event
	closeWg            sync.WaitGroup
}

// NewRepoService create repo service:
// - signal dangling artifacts at startup/repo update
func NewRepoService(log ports.Logger, bus ports.EventBus, walk disk.FilepathWalk, repoRepository domain.RepoRepository) *RepoService {
	log = log.With(slog.String("entity", "RepoService"))
	s := &RepoService{
		log:                log,
		bus:                bus,
		walk:               walk,
		repoRepository:     repoRepository,
		chTopicRepoUpdated: bus.Sub(ports.TopicRepoUpdated),
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
	s.bus.Unsub(s.chTopicRepoUpdated)
	s.closeWg.Wait()
}

func (s *RepoService) background() {
	for {
		select {
		case ids, ok := <-s.chTopicRepoUpdated:
			if !ok {
				return
			}
			for _, id := range ids {
				s.checkRepoById(id)
			}
		}
	}
}

func (s *RepoService) checkRepoById(repoID models.RepoID) {
	repo, err := s.repoRepository.FindByID(repoID, ports.WithRelationship(true))
	if err != nil {
		s.log.Error("unable to find repo", slog.Any("repoID", repoID), slog.Any("err", err))
		return
	}
	s.checkRepoStorage(repo)
}

func (s *RepoService) checkRepoStorage(repo *models.Repo) {
	log, fs := s.log.With(slog.Any("repoID", repo.RepoID)), afero.NewOsFs()
	log.Debug("check repo")

	// TODO Abstract out storage!!!!
	storage := repo.Storage
	exist, _ := afero.DirExists(fs, storage)
	if !exist {
		log.Error("storage not found", slog.String("storage", storage))
		return
	}

	//
	// Check dangling repo's artifacts
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
		if repo.Artifacts.HasArtifactID(artifactID) {
			return true, nil
		}

		// This is dangling artifact
		// It presents in repo storage but not in database
		// There might be few reasons for that:
		// - we just starting up
		// - it was manually written to storage
		// - it was written by other instance or means
		s.bus.Pub(ports.TopicDanglingRepoArtifact, ports.Event{repo.RepoID, artifactID})
		return true, nil
	})
}
