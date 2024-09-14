package adapters

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"github.com/oklog/ulid/v2"
	"xorm.io/xorm"
)

type BasicArtifactStorageAdapter struct {
	log    *ports.Logger
	engine *xorm.Engine
}

func NewBasicArtifactStorageAdapter(log *ports.Logger, engine *xorm.Engine) (*BasicArtifactStorageAdapter, error) {
	log = log.With(slog.String("entity", "BasicArtifactStorageAdapter"))
	s := &BasicArtifactStorageAdapter{
		log:    log,
		engine: engine,
	}

	return s, nil
}

func (s *BasicArtifactStorageAdapter) NewArtifact(repo *models.Repo, artifacts []string, id models.ArtifactID) (models.ArtifactID, time.Time, error) {
	lib.Assert(repo != nil)
	lib.Assert(len(artifacts) >= 1)
	log := s.log
	storage := repo.Storage
	if id == "" {
		id = models.ArtifactID(ulid.Make().String())
	}
	log = log.With(slog.String("repo", repo.Name), slog.String("id", string(id)))
	log.Info("add artifacts", slog.String("storage", storage), slog.Any("files", artifacts))

	if !lib.IsDirectoryExist(storage) {
		return "", time.Time{}, errors.ErrNoSuchDirectory{Path: storage}
	}

	dest := filepath.Join(storage, string(id))
	if lib.IsDirectoryExist(dest) {
		return "", time.Time{}, errors.ErrArtifactAlreadyExists{Path: dest}
	}
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return "", time.Time{}, err
	}

	input := repo.Input
	for _, fileName := range artifacts {
		// The input mist be sanitized already!!!
		lib.Assert(lib.IsSecureFileName(fileName))
		// Using input, filename and id to detect path withing artifact
		name := fileName
		name = strings.TrimPrefix(name, input)
		name = strings.TrimPrefix(name, string(os.PathSeparator))
		name = strings.TrimPrefix(name, string(id)+string(os.PathSeparator))
		dir, file := filepath.Split(name)
		dest := filepath.Join(dest, dir)
		if dir != "" {
			if err := os.MkdirAll(dest, os.ModePerm); err != nil {
				return "", time.Time{}, err
			}
		}
		if err := os.Rename(fileName, filepath.Join(dest, file)); err != nil {
			return "", time.Time{}, err
		}
	}

	// Optional create file _createdAt.txt containing epoch time.
	// It can be part of artifacts as well.
	// In such case the creation time would be preserved by checksum file.
	// Can be created by ```date +%s > _createdAt.txt```
	now := time.Now().Unix()
	file := filepath.Join(dest, "_createdAt.txt")
	if err := lib.CreateFile(file, fmt.Sprintf("%v", now)); lib.NoSuchFile(file) && err != nil {
		log.Warn("unable to create", slog.String("file", file), slog.Any("err", err))
	}

	// Read back creation time
	a, err := os.ReadFile(file)
	if err != nil {
		log.Warn("unable to read", slog.String("file", file), slog.Any("err", err))
	}
	// Once external creation time might be created with tailing \n or even more
	// parse only leading digits and ignore rest
	t, err := strconv.ParseInt(lib.LeadingDigits(string(a)), 10, 64)
	if err != nil {
		log.Warn("unable convert creation time", slog.Any("err", err))
	}
	createdAt := time.Unix(t, 0)

	return id, createdAt, nil
}

func (s *BasicArtifactStorageAdapter) Close() {
	log := s.log
	log.Info("closing")
}
