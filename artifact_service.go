package swamp

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/infra/config"
	"github.com/cloudcopper/swamp/infra/disk"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/lib/types"
	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
)

// ArtifactService listeing eventbus for next events:
//   - repo-updated - to maintain internal list of repos
//   - input-file-modified - to check if the file is checksum belonging to any of known repos,
//     and if so, then create new artifact by checksum file
//   - dangling-repo-artifact - to check/add dangling repo artifact
type ArtifactService struct {
	log                         ports.Logger
	bus                         ports.EventBus
	artifactStorage             ports.ArtifactStorage
	repositories                domain.Repositories
	chTopicRepoUpdated          chan ports.Event
	chTopicInputFileModified    chan ports.Event
	chTopicDanglingRepoArtifact chan ports.Event
	closeWg                     sync.WaitGroup
}

func NewArtifactService(log ports.Logger, bus ports.EventBus, artifactStorage ports.ArtifactStorage, repositories domain.Repositories) (*ArtifactService, error) {
	log = log.With(slog.String("entity", "ArtifactService"))

	if _, err := repositories.Repo().FindAll(); err != nil {
		return nil, err
	}

	s := &ArtifactService{
		log:                         log,
		bus:                         bus,
		artifactStorage:             artifactStorage,
		repositories:                repositories,
		chTopicRepoUpdated:          bus.Sub(ports.TopicRepoUpdated),
		chTopicInputFileModified:    bus.Sub(ports.TopicInputFileModified),
		chTopicDanglingRepoArtifact: bus.Sub(ports.TopicDanglingRepoArtifact),
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
	s.bus.Unsub(s.chTopicDanglingRepoArtifact)
	s.bus.Unsub(s.chTopicInputFileModified)
	s.bus.Unsub(s.chTopicRepoUpdated)
	s.closeWg.Wait()
}

func (s *ArtifactService) background() {
	log := s.log

	repos, err := s.repositories.Repo().FindAll()
	if err != nil {
		log.Error("unable to read all repos", slog.Any("err", err))
		return
	}

	timerExpired := time.NewTimer(config.TimerExpiredStart)
	defer timerExpired.Stop()

	timerBroken := time.NewTimer(config.TimerBrokenStart)
	defer timerBroken.Stop()
	knownArtifacts := []*models.Artifact{}

	for {
		select {
		case event, ok := <-s.chTopicRepoUpdated:
			_ = event
			if !ok {
				return
			}
			repos, err = s.repositories.Repo().FindAll()
			if err != nil {
				log.Error("unable to update all repos", slog.Any("err", err))
				return
			}
		case event, ok := <-s.chTopicInputFileModified:
			if !ok {
				return
			}
			path := event[0]
			s.checkInputFile(repos, afero.NewOsFs(), path)
		case event, ok := <-s.chTopicDanglingRepoArtifact:
			if !ok {
				return
			}
			repoID, artifactID := event[0], event[1]
			s.checkRepoArtifact(repoID, artifactID)
		case _, ok := <-timerExpired.C:
			if !ok {
				return
			}
			// Remove already expired artifacts
			// and then update expired artifacts.
			// That allows expired artifact to stay
			// in db at least one cycle prior being removed.
			// The limit defines how many expired artifacts
			// per cycle can be removed.
			limit := config.TimerExpiredLimit
			s.removeExpiredArtifacts(limit)
			now := time.Now().UTC().Unix()
			s.markExpiredArtifacts(now)

			timerExpired.Reset(config.TimerExpiredInterval)
		case _, ok := <-timerBroken.C:
			if !ok {
				return
			}
			limit := config.TimerBrokenLimit
			s.removeBrokenArtifacts(limit)
			knownArtifacts = s.checkBrokenArtifacts(limit, knownArtifacts)
			timerBroken.Reset(config.TimerBrokenInterval)
		}
	}
}

func (s *ArtifactService) checkInputFile(repos []*models.Repo, fs ports.FS, path string) {
	log := s.log.With(slog.String("path", path))
	log.Debug("detect modified")

	artifactStorage := s.artifactStorage

	for _, repo := range repos {
		// Check the path belongs to repo
		if !strings.HasPrefix(path, repo.Input) {
			continue
		}
		log := log.With(slog.Any("repoID", repo.ID))
		log.Debug("path match repo")

		// Check the path is a good checksum
		checksum, goodFiles, badFiles, err := adapters.CheckChecksum(log, fs, path)
		if err == errors.ErrIsNotChecksumFile {
			continue
		}
		if err != nil {
			log.Error("unable to verify checksum file", slog.Any("goodFiles", goodFiles), slog.Any("badFile", badFiles), slog.Any("err", err))
			break
		}
		log.Info("checksum file verified", slog.Any("goodFiles", goodFiles))

		// Try to get artifact metas
		metas := map[string]string{}
		for _, f := range goodFiles {
			if !adapters.IsMetaFile(f) {
				continue
			}
			meta, err := adapters.ParseMetaFile(log, fs, f)
			if err != nil {
				continue
			}
			for k, v := range meta {
				metas[k] = v
			}
		}

		// Create new artifacts
		artifacts := append(goodFiles, path)
		id := lib.GetFirstSubdir(repo.Input, path)
		info, err := artifactStorage.NewArtifact(repo.Input, repo.Storage, id, artifacts)
		if err != nil {
			log.Error("unable to create new artifacts", slog.Any("err", err))
		}
		log.Info("new artifact created", slog.Any("artifactID", info.ID))

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
				exist, _ := afero.DirExists(fs, name)
				if !exist {
					continue
				}
			}
			if err := fs.RemoveAll(name); err != nil {
				log.Warn("unable to remove input artifact", slog.Any("err", err), slog.String("name", name))
			}
		}

		// Covert artifact meta
		meta := models.ArtifactMetas{}
		for k, v := range metas {
			meta = append(meta, &models.ArtifactMeta{
				RepoID:     repo.ID,
				ArtifactID: info.ID,
				Key:        k,
				Value:      v,
			})
		}

		// Insert artifact record
		createdAt := info.CreatedAt
		expiredAt := createdAt + int64(repo.Retention/1000000000)
		state := vo.ArtifactIsOK
		if expiredAt != createdAt && expiredAt < time.Now().UTC().Unix() {
			state |= vo.ArtifactIsExpired
		}
		artifact := &models.Artifact{
			ID:        info.ID,
			RepoID:    repo.ID,
			Storage:   repo.Storage,
			Size:      types.Size(info.Size),
			State:     state,
			CreatedAt: info.CreatedAt,
			ExpiredAt: expiredAt,
			Checksum:  checksum,
			Meta:      meta,
		}
		if err := s.repositories.Artifact().Create(artifact); err != nil {
			log.Error("unable create artifact record", slog.Any("artifactID", artifact.ID), slog.Any("err", err))
		}
		s.bus.Pub(ports.TopicArtifactUpdated, ports.Event{artifact.RepoID, artifact.ID})
		return
	}
}

// The checkRepoArtifact checks the artifact inside repo storage.
// If it dangling, it creates new artifact model.
func (s *ArtifactService) checkRepoArtifact(repoID models.RepoID, artifactID models.ArtifactID) {
	log := s.log.With(slog.Any("repoID", repoID), slog.Any("artifactID", artifactID))
	repo, err := s.repositories.Repo().FindByID(repoID)
	if err != nil {
		log.Error("unable to fine repo by id", slog.Any("err", err))
		return
	}

	loc := filepath.Join(repo.Storage, artifactID)
	size, createdAt, checksum, err := s.verifyArtifact(afero.NewOsFs(), loc)
	if err != nil {
		log.Error("unable to verify aritfact", slog.Any("err", err))
		s.bus.Pub(ports.TopicBrokenRepoArtifact, ports.Event{repoID, artifactID})
		return
	}

	artifact, err := s.repositories.Artifact().FindByID(repoID, artifactID)
	if err != nil && !errors.Is(err, ports.ErrRecordNotFound) {
		log.Error("unable to find artifact", slog.Any("err", err))
		return
	}
	if artifact.ID == models.EmptyArtifactID {
		log.Info("dangling artifact")
		expiredAt := createdAt + int64(repo.Retention/1000000000)
		state := vo.ArtifactIsOK
		if expiredAt != createdAt && expiredAt < time.Now().UTC().Unix() {
			state |= vo.ArtifactIsExpired
		}
		artifact.ID = artifactID
		artifact.RepoID = repoID
		artifact.Storage = repo.Storage
		artifact.Size = types.Size(size)
		artifact.Checksum = checksum
		artifact.State = state
		artifact.CreatedAt = createdAt
		artifact.ExpiredAt = expiredAt
		if err := s.repositories.Artifact().Create(artifact); err != nil {
			log.Error("unable create artifact record", slog.Any("err", err))
			return
		}
		log.Info("artifact re-created")
		s.bus.Pub(ports.TopicArtifactUpdated, ports.Event{artifact.RepoID, artifact.ID})
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
func (s *ArtifactService) verifyArtifact(fs ports.FS, location string) (int64, int64, string, error) {
	checksumFile, files := "", []string{}

	w := disk.NewFilepathWalk(fs)
	w.Walk(location, func(name string, err error) (bool, error) {
		if err != nil {
			s.log.Error("walk error", slog.String("location", location), slog.String("name", name), slog.Any("err", err))
			return true, nil
		}
		exist, _ := afero.DirExists(fs, name)
		if exist {
			return true, nil
		}
		if adapters.IsChecksumFile(name) {
			if checksumFile != "" {
				s.log.Error("second checksum file detected", slog.String("checksumFile", checksumFile), slog.String("name", name))
				return true, nil
			}
			checksumFile = name
		}
		files = append(files, name)
		return true, nil
	})

	checksum, goodFiles, badFiles, err := adapters.CheckChecksum(s.log, fs, checksumFile)
	if err != nil {
		s.log.Error("unable to checksum artifact", slog.String("checksumFile", checksumFile), slog.Any("err", err))
		return 0, 0, "", errors.ErrArtifactIsBroken
	}
	if len(badFiles) > 0 {
		s.log.Error("bad files detected", slog.String("checksumFile", checksumFile), slog.Any("badFiles", badFiles))
		return 0, 0, "", errors.ErrArtifactIsBroken
	}

	if !slices.Contains(goodFiles, checksumFile) {
		s.log.Warn("checksum file is not in checksum file")
		goodFiles = append(goodFiles, checksumFile)
	}
	createdAtName := filepath.Join(filepath.Dir(checksumFile), "_createdAt.txt")
	if !slices.Contains(goodFiles, createdAtName) {
		s.log.Warn("createdAt file is not in checksum file")
		goodFiles = append(goodFiles, createdAtName)
	}

	if len(goodFiles) != len(files) {
		s.log.Error("missmatch between good and actual files", slog.Any("goodFiles", goodFiles), slog.Any("files", files))
		return 0, 0, "", errors.ErrArtifactIsBroken
	}

	slices.Sort(goodFiles)
	slices.Sort(files)
	if slices.Compare(goodFiles, files) != 0 {
		s.log.Error("different files listed in good and actual files", slog.Any("goodFiles", goodFiles), slog.Any("files", files))
		return 0, 0, "", errors.ErrArtifactIsBroken
	}

	// Read back creation time
	a, err := os.ReadFile(createdAtName)
	if err != nil {
		s.log.Warn("unable to read", slog.String("file", createdAtName), slog.Any("err", err))
	}
	// Once external creation time might be created with tailing \n or even more
	// parse only leading digits and ignore rest
	t, err := strconv.ParseInt(lib.LeadingDigits(string(a)), 10, 64)
	if err != nil {
		s.log.Warn("unable convert creation time", slog.Any("err", err))
	}
	createdAt := t

	size := int64(0)
	for _, file := range goodFiles {
		size += lib.FileSize(fs, file)
	}

	return size, createdAt, checksum, nil
}

func (s *ArtifactService) markExpiredArtifacts(now int64) {
	log := s.log
	artifacts, err := s.repositories.Artifact().FindAllTimeExpired(now)
	if err != nil {
		log.Error("unable fetch all now expired artifacts", slog.Any("err", err))
		return
	}

	for _, artifact := range artifacts {
		lib.Assert(!artifact.State.IsExpired())
		log.Info("mark artifact expired", slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ID))
		artifact.State |= vo.ArtifactIsExpired
		err := s.repositories.Artifact().Update(artifact)
		if err != nil {
			log.Error("unable set artifact expired", slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ID), slog.Any("err", err))
		}
		s.bus.Pub(ports.TopicArtifactUpdated, ports.Event{artifact.RepoID, artifact.ID})
	}
}

func (s *ArtifactService) removeExpiredArtifacts(limit int) {
	log := s.log
	artifacts, err := s.repositories.Artifact().FindAllStatusExpired(ports.Limit(limit))
	if err != nil {
		log.Error("unable fetch all expired artifacts", slog.Any("err", err))
		return
	}

	for _, artifact := range artifacts {
		log := log.With(slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ID))
		log.Info("remove expired artifact")
		lib.Assert(artifact.State.IsExpired())
		if err := s.artifactStorage.RemoveArtifact(artifact.Storage, artifact.ID); err != nil {
			log.Error("artifact path remove failed", slog.Any("storage", artifact.Storage), slog.Any("artifactID", artifact.ID), slog.Any("err", err))
		}
		if err := s.repositories.Artifact().Delete(artifact); err != nil {
			log.Error("artifact model delete failed", slog.Any("err", err))
		}
	}
}

func (s *ArtifactService) checkBrokenArtifacts(limit int, artifacts []*models.Artifact) []*models.Artifact {
	log := s.log
	if len(artifacts) == 0 {
		var err error
		artifacts, err = s.repositories.Artifact().FindAllStatusNotBroken()
		if err != nil {
			log.Error("unable fetch all not broken artifacts", slog.Any("err", err))
			return nil
		}
	}

	for x := 0; x < limit && len(artifacts) > 0; x++ {
		artifact := artifacts[0]
		artifacts = artifacts[1:]
		lib.Assert(!artifact.State.IsBroken())

		log := log.With(slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ID))
		loc := filepath.Join(artifact.Storage, artifact.ID)
		size, createdAt, checksum, err := s.verifyArtifact(afero.NewOsFs(), loc)
		is_broken := false
		if err != nil {
			log.Error("unable verify artifact", slog.Any("err", err))
			is_broken = true
		}
		if err == nil && size != int64(artifact.Size) {
			log.Error("artifact size dont match", slog.Any("size", size), slog.Any("artifact.Size", artifact.Size))
			is_broken = true
		}
		if err == nil && createdAt != artifact.CreatedAt {
			log.Error("artifact createdAt dont match", slog.Any("createdAt", createdAt), slog.Any("artifact.CreatedAt", artifact.CreatedAt))
			is_broken = true
		}
		if err == nil && checksum != artifact.Checksum {
			log.Error("artifact checksum dont match", slog.Any("checksum", checksum), slog.Any("artifact.Checksum", artifact.Checksum))
			is_broken = true
		}

		if is_broken {
			log.Warn("mark artifact broken", slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ID))
			artifact.State |= vo.ArtifactIsBroken
			err := s.repositories.Artifact().Update(artifact)
			if err != nil {
				log.Error("unable set artifact broken", slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ID), slog.Any("err", err))
			}
			s.bus.Pub(ports.TopicArtifactUpdated, ports.Event{artifact.RepoID, artifact.ID})
		}
	}

	return artifacts
}

func (s *ArtifactService) removeBrokenArtifacts(limit int) {
	log := s.log
	artifacts, err := s.repositories.Artifact().FindAllStatusBroken(ports.Limit(limit))
	if err != nil {
		log.Error("unable fetch all expired artifacts", slog.Any("err", err))
		return
	}

	for _, artifact := range artifacts {
		log := log.With(slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ID))
		log.Info("process broken artifact")
		lib.Assert(artifact.State.IsBroken())
		path := filepath.Join(artifact.Storage, artifact.ID)

		// detect the location for artifact to be moved to (or removed)
		repo, err := s.repositories.Repo().FindByID(artifact.RepoID)
		if err != nil {
			log.Error("unable fetch repo model", slog.Any("err", err))
			continue
		}

		broken := repo.Broken
		if broken == "" {
			continue
		}
		remove := false
		if broken == "/dev/null" {
			remove = true
		}

		if remove {
			log.Info("remove broken artifact", slog.Any("path", path))
			if err := os.RemoveAll(path); err != nil {
				log.Error("artifact path remove failed", slog.Any("path", path), slog.Any("err", err))
			}
		}
		if !remove {
			newpath := filepath.Join(broken, strings.Join([]string{repo.ID, artifact.ID}, "-"))
			log.Info("move broken artifact", slog.Any("path", path), slog.Any("newpath", newpath))
			if err := os.Rename(path, newpath); err != nil {
				log.Error("artifact path move failed", slog.Any("path", path), slog.Any("newpath", newpath), slog.Any("err", err))
			}
		}
		if err := s.repositories.Artifact().Delete(artifact); err != nil {
			log.Error("artifact model delete failed", slog.Any("err", err))
		}
	}
}
