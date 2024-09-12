package adapters

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"github.com/oklog/ulid/v2"
	"xorm.io/xorm"
)

type BasicArtifactsStorageAdapter struct {
	log    *ports.Logger
	engine *xorm.Engine
}

func NewBasicArtifactsStorageAdapter(log *ports.Logger, engine *xorm.Engine) (*BasicArtifactsStorageAdapter, error) {
	log = log.With(slog.String("entity", "BasicArtifactsStorageAdapter"))
	s := &BasicArtifactsStorageAdapter{
		log:    log,
		engine: engine,
	}

	return s, nil
}

func (s *BasicArtifactsStorageAdapter) NewArtifacts(repo *domain.Repo, artifacts []string, id domain.ArtifactID) (domain.ArtifactID, error) {
	lib.Assert(repo != nil)
	lib.Assert(len(artifacts) >= 1)
	log := s.log
	storage := repo.Storage
	if id == "" {
		id = domain.ArtifactID(ulid.Make().String())
	}
	log = log.With(slog.String("repo", repo.Name), slog.String("id", string(id)))
	log.Info("add artifacts", slog.String("storage", storage), slog.Any("files", artifacts))

	if !lib.IsDirectoryExist(storage) {
		return "", errors.ErrNoSuchDirectory{Path: storage}
	}

	dest := filepath.Join(storage, string(id))
	if lib.IsDirectoryExist(dest) {
		return "", errors.ErrArtifactAlreadyExists{Path: dest}
	}
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return "", err
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
				return "", err
			}
		}
		if err := os.Rename(fileName, filepath.Join(dest, file)); err != nil {
			return "", err
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

	/*
		a, err := os.ReadFile(file)
		if err != nil {
			log.Warn("unable to read", slog.String("file", file), slog.Any("err", err))
		}
		now, err = strconv.ParseInt(string(a), 10, 64)
		if err != nil {
			log.Warn("unable convert creation time", slog.Any("err", err))
		}

		// TODO Now we have to update database
		_ = now
	*/

	return id, nil
}

func (s *BasicArtifactsStorageAdapter) Close() {
	log := s.log
	log.Info("closing")
}
