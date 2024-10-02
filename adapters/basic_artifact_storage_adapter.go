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
)

type BasicArtifactStorageAdapter struct {
	log ports.Logger
	db  ports.DB
}

func NewBasicArtifactStorageAdapter(log ports.Logger, db ports.DB) (*BasicArtifactStorageAdapter, error) {
	log = log.With(slog.String("entity", "BasicArtifactStorageAdapter"))
	s := &BasicArtifactStorageAdapter{
		log: log,
		db:  db,
	}

	return s, nil
}

func (s *BasicArtifactStorageAdapter) NewArtifact(repo *models.Repo, id models.ArtifactID, artifacts []string) (models.ArtifactID, int64, int64, error) {
	lib.Assert(repo != nil)
	lib.Assert(len(artifacts) >= 1)
	log := s.log
	storage := repo.Storage
	if id == "" {
		id = ulid.Make().String()
	}
	log = log.With(slog.Any("repoID", repo.ID), slog.String("artifactID", string(id)))
	log.Info("add artifacts", slog.String("storage", storage), slog.Any("files", artifacts))

	if !lib.IsDirectoryExist(storage) {
		return "", 0, 0, lib.ErrNoSuchDirectory{Path: storage}
	}

	dest := filepath.Join(storage, string(id))
	if lib.IsDirectoryExist(dest) {
		return "", 0, 0, errors.ErrArtifactAlreadyExists{Path: dest}
	}
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return "", 0, 0, err
	}

	input := repo.Input
	size := int64(0)
	for _, fileName := range artifacts {
		// The input mist be sanitized already!!!
		lib.Assert(lib.IsSecureFileName(fileName))
		// Using input, filename and id to detect path withing artifact
		name := fileName
		name = strings.TrimPrefix(name, input)
		name = strings.TrimPrefix(name, string(os.PathSeparator))
		name = strings.TrimPrefix(name, id+string(os.PathSeparator))
		dir, file := filepath.Split(name)
		dest := filepath.Join(dest, dir)
		if dir != "" {
			if err := os.MkdirAll(dest, os.ModePerm); err != nil {
				return "", 0, 0, err
			}
		}
		newpath := filepath.Join(dest, file)
		if err := os.Rename(fileName, newpath); err != nil {
			return "", 0, 0, err
		}
		size = size + lib.FileSize(newpath)
	}

	// Optional create file _createdAt.txt containing epoch time.
	// It can be part of artifacts as well.
	// In such case the creation time would be preserved by checksum file.
	// Can be created by ```date +%s > _createdAt.txt```
	now := time.Now().UTC().Unix()
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
	createdAt := t

	return id, size, createdAt, nil
}

func (s *BasicArtifactStorageAdapter) Close() {
	log := s.log
	log.Info("closing")
}
