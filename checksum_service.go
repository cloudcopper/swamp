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
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"xorm.io/xorm"
)

type ChecksumService struct {
	log              *ports.Logger
	engine           *xorm.Engine
	watcher          InputWatcherService
	artifactsStorage ports.ArtifactsStorage
	closeWg          sync.WaitGroup
}

func NewChecksumService(log *ports.Logger, engine *xorm.Engine, watcher InputWatcherService, artifactsStorage ports.ArtifactsStorage) (*ChecksumService, error) {
	log = log.With(slog.String("entity", "ChecksumService"))

	if _, err := FindAll[domain.Repo](engine); err != nil {
		return nil, err
	}

	s := &ChecksumService{
		log:              log,
		engine:           engine,
		watcher:          watcher,
		artifactsStorage: artifactsStorage,
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
	engine := s.engine
	modified := s.watcher.GetChanModified()
	artifactsStorage := s.artifactsStorage

	repos, err := FindAll[domain.Repo](engine)
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
			id := getFirstSubdir(repo.Input, path)
			artifactId, err := artifactsStorage.NewArtifacts(repo, artifacts, domain.ArtifactID(id))
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
				dir := getFirstSubdir(input, file)
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

			//
			// TODO Now we have to update add artifact record
			//

			break
		}
	}
}

// The gitFirstSubdir returns first directory name after input
// Example:
// input is '/mnt/input/project'
// if path is '/mnt/input/project/1234.crc' then return is ”
// if path is '/mnt/input/project/rel-4.2.2/1234.crc' then return is 'rel-4.2.2'
func getFirstSubdir(input, path string) string {
	lib.Assert(strings.HasPrefix(path, input))
	a := strings.Split(strings.TrimLeft(strings.TrimPrefix(path, input), string(os.PathSeparator)), string(os.PathSeparator))
	lib.Assert(len(a) >= 1)
	lib.Assert(a[0] != "")
	dir := a[0]
	if len(a) <= 1 {
		dir = ""
	}

	return dir
}
