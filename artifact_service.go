package swamp

import (
	"fmt"
	"log/slog"
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
	"github.com/oklog/ulid/v2"
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
	storageFs                   ports.FS
	brokenFs                    ports.FS
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
		storageFs:                   afero.NewOsFs(),
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
			// TODO Can watcher run over ports.FS?
			// TODO Should we change watcher to some poll/scan mode watcher which would ran well over ports.FS?
			// TODO Event should have inputFS!!!
			s.checkInputFile(repos, s.storageFs, path)
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

// The checkInputFile from background where any changed in noticed.
// The repos has all known cached atm repo definitions.
// The fs is a file system where the even was detected.
// The path is the full path of changes detected.
func (s *ArtifactService) checkInputFile(repos []*models.Repo, f ports.FS, path string) error {
	for _, repo := range repos {
		if strings.HasPrefix(path, repo.Input) { // Check the path belongs to repo.Input
			return s.checkRepoInput(repo, f, path)
		}
	}
	return errors.ErrNotMatchRepoInput
}
func (s *ArtifactService) checkRepoInput(repo *models.Repo, f ports.FS, checksumFile string) error {
	log := s.log.With(slog.Any("checksumFile", checksumFile), slog.Any("repoID", repo.RepoID))

	// Check the path is a good checksum
	da := checksumDiskArtifact(log, f, checksumFile)
	if da.checksumError != nil {
		return da.checksumError
	}
	log.Info("checksum file verified", slog.Any("files.Good", da.files.Good))

	// get artifact meta and files
	meta := da.getArtifactMeta(log)
	files := da.getArtifactFiles(log)

	// Create new artifacts
	artifacts := da.files.Good
	artifactID := lib.GetFirstSubdir(repo.Input, da.checksumFile)
	if artifactID == "" {
		artifactID = ulid.Make().String()
	}
	log.Info("new artifact", slog.Any("artifactID", artifactID))

	info, err := s.artifactStorage.NewArtifact(f, repo.Input, artifacts, repo.Storage, artifactID)
	if err != nil {
		log.Error("unable to create new artifacts", slog.Any("err", err))
	}

	// Cleanup input artifacts
	cleanInputArtifacts(log, f, repo.Input, artifacts)

	// Insert artifact record
	createdAt := info.CreatedAt
	expiredAt := createdAt + int64(repo.Retention/1000000000)
	state := vo.ArtifactIsOK
	artifact := &models.Artifact{
		ArtifactID: artifactID,
		RepoID:     repo.RepoID,
		Storage:    repo.Storage,
		Size:       types.Size(info.Size),
		State:      state,
		CreatedAt:  info.CreatedAt,
		ExpiredAt:  expiredAt,
		Checksum:   string(da.checksum),
		Meta:       meta,
		Files:      files,
	}
	if err := s.repositories.Artifact().Create(artifact); err != nil {
		log.Error("unable create artifact record", slog.Any("artifactID", artifact.ArtifactID), slog.Any("err", err))
	}
	s.bus.Pub(ports.TopicArtifactUpdated, ports.Event{artifact.RepoID, artifact.ArtifactID})
	return nil
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
	da, err := s.verifyArtifactLocation(loc)
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
	if artifact.ArtifactID == models.EmptyArtifactID {
		log.Info("dangling artifact")
		expiredAt := da.createdAt + int64(repo.Retention/1000000000)
		state := vo.ArtifactIsOK
		if expiredAt != da.createdAt && expiredAt < time.Now().UTC().Unix() {
			state |= vo.ArtifactIsExpired
		}

		// get artifact meta and files
		meta := da.getArtifactMeta(log)
		files := da.getArtifactFiles(log)

		artifact := &models.Artifact{
			ArtifactID: artifactID,
			RepoID:     repoID,
			Storage:    repo.Storage,
			Size:       types.Size(da.size),
			Checksum:   string(da.checksum),
			State:      state,
			CreatedAt:  da.createdAt,
			ExpiredAt:  expiredAt,
			Meta:       meta,
			Files:      files,
		}

		if err := s.repositories.Artifact().Create(artifact); err != nil {
			log.Error("unable create artifact record", slog.Any("err", err))
			return
		}
		log.Info("artifact re-created")
		s.bus.Pub(ports.TopicArtifactUpdated, ports.Event{artifact.RepoID, artifact.ArtifactID})
		return
	}
	if artifact.ArtifactID != artifactID {
		// This would be some serious issue
		// We expect to read from artifact repository
		// either requested ID not empty
		log.Error("wrong artifact found", slog.Any("unexpected artifact id", artifact.ArtifactID))
		return
	}
	if artifact.CreatedAt == da.createdAt && artifact.Checksum != da.checksum {
		log.Error("tampered artifact", slog.Any("original checksum", artifact.Checksum), slog.Any("checksum", da.checksum))
		s.bus.Pub(ports.TopicBrokenRepoArtifact, ports.Event{repoID, artifactID})
		return
	}
	if artifact.CreatedAt != da.createdAt && artifact.Checksum == da.checksum {
		log.Warn("reuploaded artifact", slog.Any("originally createdAt", artifact.CreatedAt), slog.Any("createdAt", da.createdAt))
		// Do nothing - keep original details
		return
	}
}

// The verifyArtifactLocation check the location artifact files
func (s *ArtifactService) verifyArtifactLocation(location string) (*diskArtifact, error) {
	log, f := s.log, s.storageFs

	// Scan disk artifact
	da := walkDiskArtifact(log, f, location)

	// Verify artifact state
	if da.checksumError != nil {
		log.Error("unable to checksum artifact", slog.String("checksumFile", da.checksumFile), slog.Any("err", da.checksumError))
		return da, errors.ErrArtifactIsBroken
	}
	if len(da.files.Bad) > 0 {
		log.Error("bad files detected", slog.String("checksumFile", da.checksumFile), slog.Any("files.Bad", da.files.Bad))
		return da, errors.ErrArtifactIsBroken
	}
	if len(da.files.Good) != len(da.allFiles) {
		log.Error("missmatch between good and actual files", slog.Any("files.Good", da.files.Good), slog.Any("files", da.allFiles))
		return da, errors.ErrArtifactIsBroken
	}
	if slices.Compare(da.files.Good, da.allFiles) != 0 {
		log.Error("different files listed in good and actual files", slog.Any("files.Good", da.files.Good), slog.Any("files", da.allFiles))
		return da, errors.ErrArtifactIsBroken
	}

	return da, nil
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
		log.Info("mark artifact expired", slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ArtifactID))
		artifact.State |= vo.ArtifactIsExpired
		err := s.repositories.Artifact().Update(artifact)
		if err != nil {
			log.Error("unable set artifact expired", slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ArtifactID), slog.Any("err", err))
		}
		s.bus.Pub(ports.TopicArtifactUpdated, ports.Event{artifact.RepoID, artifact.ArtifactID})
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
		log := log.With(slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ArtifactID))
		log.Info("remove expired artifact")
		lib.Assert(artifact.State.IsExpired())
		if err := s.artifactStorage.RemoveArtifact(artifact.Storage, artifact.ArtifactID); err != nil {
			log.Error("artifact path remove failed", slog.Any("storage", artifact.Storage), slog.Any("artifactID", artifact.ArtifactID), slog.Any("err", err))
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
		s.checkBrokenArtifact(artifact)
	}
	return artifacts
}

func (s *ArtifactService) checkBrokenArtifact(artifact *models.Artifact) {
	log := s.log.With(slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ArtifactID))
	loc := filepath.Join(artifact.Storage, artifact.ArtifactID)
	da, err := s.verifyArtifactLocation(loc)
	is_broken := false
	if err != nil {
		log.Error("unable verify artifact", slog.Any("err", err))
		is_broken = true
	}
	if err == nil && da.size != int64(artifact.Size) {
		log.Error("artifact size dont match", slog.Any("size", da.size), slog.Any("artifact.Size", artifact.Size))
		is_broken = true
	}
	if err == nil && da.createdAt != artifact.CreatedAt {
		log.Error("artifact createdAt dont match", slog.Any("createdAt", da.createdAt), slog.Any("artifact.CreatedAt", artifact.CreatedAt))
		is_broken = true
	}
	if err == nil && da.checksum != artifact.Checksum {
		log.Error("artifact checksum dont match", slog.Any("checksum", da.checksum), slog.Any("artifact.Checksum", artifact.Checksum))
		is_broken = true
	}
	if is_broken {
		log.Warn("mark artifact broken", slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ArtifactID))
		artifact.State |= vo.ArtifactIsBroken
		err := s.repositories.Artifact().Update(artifact)
		if err != nil {
			log.Error("unable set artifact broken", slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ArtifactID), slog.Any("err", err))
		}
		s.bus.Pub(ports.TopicArtifactUpdated, ports.Event{artifact.RepoID, artifact.ArtifactID})
	}
}

func (s *ArtifactService) removeBrokenArtifacts(limit int) {
	log := s.log
	artifacts, err := s.repositories.Artifact().FindAllStatusBroken(ports.Limit(limit))
	if err != nil {
		log.Error("unable fetch all expired artifacts", slog.Any("err", err))
		return
	}

	for _, artifact := range artifacts {
		log := log.With(slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ArtifactID))
		log.Info("process broken artifact")
		lib.Assert(artifact.State.IsBroken())
		path := filepath.Join(artifact.Storage, artifact.ArtifactID)

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
			if err := s.storageFs.RemoveAll(path); err != nil {
				log.Error("artifact path remove failed", slog.Any("path", path), slog.Any("err", err))
			}
		}
		if !remove {
			newpath := filepath.Join(broken, fmt.Sprintf("%v-%v", repo.RepoID, artifact.ArtifactID))
			log.Info("move broken artifact", slog.Any("path", path), slog.Any("newpath", newpath))
			if err := lib.MoveFile(s.storageFs, path, s.brokenFs, newpath); err != nil {
				log.Error("artifact path move failed", slog.Any("path", path), slog.Any("newpath", newpath), slog.Any("err", err))
			}
		}
		if err := s.repositories.Artifact().Delete(artifact); err != nil {
			log.Error("artifact model delete failed", slog.Any("err", err))
		}
	}
}

func cleanInputArtifacts(log ports.Logger, f ports.FS, input string, artifacts []string) {
	log.Info("cleanup input artifacts")
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
			exist, _ := afero.DirExists(f, name)
			if !exist {
				continue
			}
		}
		if err := f.RemoveAll(name); err != nil {
			log.Warn("unable to remove input artifact", slog.Any("err", err), slog.String("name", name))
		}
	}
}

type diskArtifact struct {
	fs            ports.FS
	location      string
	allFiles      []string
	files         ports.CheckedFiles
	checksum      string
	checksumFile  string
	createdAt     int64
	createdAtFile string
	size          int64
	checksumError error
}

// The walkDiskArtifact create diskArtifact object base on files in the location
// The returned diskArtifact.checksumError will not be nil if checksum had a problems
func walkDiskArtifact(log ports.Logger, f ports.FS, location string) *diskArtifact {
	da := &diskArtifact{
		fs:       f,
		location: location,
	}

	// Detect checksum file and all files in location
	w := disk.NewFilepathWalk(f)
	w.Walk(location, func(name string, err error) (bool, error) {
		if err != nil {
			log.Error("walk error", slog.String("location", location), slog.String("name", name), slog.Any("err", err))
			return true, nil
		}
		exist, _ := afero.DirExists(f, name)
		if exist {
			return true, nil
		}
		if adapters.IsChecksumFile(name) {
			if da.checksumFile != "" {
				log.Error("second checksum file detected", slog.String("checksumFile", da.checksumFile), slog.String("name", name))
				return true, nil
			}
			da.checksumFile = name
		}
		da.allFiles = append(da.allFiles, name)
		return true, nil
	})

	da.processChecksumFile(log)
	return da
}

func checksumDiskArtifact(log ports.Logger, f ports.FS, checksumFile string) *diskArtifact {
	da := &diskArtifact{
		fs:           f,
		checksumFile: checksumFile,
	}

	da.processChecksumFile(log)
	return da
}

func (da *diskArtifact) processChecksumFile(log ports.Logger) error {
	// Detect checksum, good files and bad files in location
	da.checksum, da.files, da.checksumError = adapters.CheckChecksum(log, da.fs, da.checksumFile)

	// If needed, add checksum file and _createdAt.txt to good files
	if !slices.Contains(da.files.Good, da.checksumFile) {
		log.Warn("checksum file is not in checksum file")
		da.files.Good = append(da.files.Good, da.checksumFile)
	}
	da.createdAtFile = filepath.Join(filepath.Dir(da.checksumFile), "_createdAt.txt")
	if lib.First(afero.Exists(da.fs, da.createdAtFile)) && !slices.Contains(da.files.Good, da.createdAtFile) {
		log.Warn("createdAt file is not in checksum file")
		da.files.Good = append(da.files.Good, da.createdAtFile)
	}
	slices.Sort(da.files.Good)
	slices.Sort(da.allFiles)

	// Read back creation time
	a, err := afero.ReadFile(da.fs, da.createdAtFile)
	if err != nil {
		log.Warn("unable to read", slog.String("file", da.createdAtFile), slog.Any("err", err))
	}
	// Once external creation time might be created with tailing \n or even more
	// parse only leading digits and ignore rest
	t, err := strconv.ParseInt(lib.LeadingDigits(string(a)), 10, 64)
	if err != nil {
		log.Warn("unable convert creation time", slog.Any("err", err))
	}
	da.createdAt = t

	// Calculate total size
	da.size = 0
	for _, file := range da.files.Good {
		da.size += lib.FileSize(da.fs, file)
	}

	return da.checksumError
}

func (da *diskArtifact) getArtifactMeta(log ports.Logger) models.ArtifactMetas {
	metas := map[string]string{}
	for _, f := range da.files.Good {
		if !adapters.IsMetaFile(f) {
			continue
		}
		meta, err := adapters.ParseMetaFile(log, da.fs, f)
		if err != nil {
			continue
		}
		for k, v := range meta {
			metas[k] = v
		}
	}

	meta := models.ArtifactMetas{}
	for k, v := range metas {
		meta = append(meta, &models.ArtifactMeta{
			Key:   k,
			Value: v,
		})
	}

	return meta
}

func (da *diskArtifact) getArtifactFiles(ports.Logger) models.ArtifactFiles {
	files := models.ArtifactFiles{}
	addFile := func(filePath string, state vo.ArtifactState) {
		fileName := strings.TrimPrefix(strings.TrimPrefix(filePath, da.location), string(filepath.Separator))
		size, err := lib.FileSize2(da.fs, filePath)
		if err != nil { // mark file as broken, if can not get it size
			state |= vo.ArtifactIsBroken
		}
		file := &models.ArtifactFile{
			Name:  fileName,
			Size:  types.Size(size),
			State: state,
		}
		files = append(files, file)
	}
	for _, f := range da.files.Good {
		addFile(f, vo.ArtifactIsOK)
	}
	for _, f := range da.files.Bad {
		addFile(f, vo.ArtifactIsBroken)
	}
	if !slices.Contains(da.files.Good, da.checksumFile) {
		addFile(da.checksumFile, vo.ArtifactIsOK)
	}

	return files
}
