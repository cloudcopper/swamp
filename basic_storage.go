package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/oklog/ulid/v2"
	"xorm.io/xorm"
)

type BasicArtifactsStorage struct {
	log    *Logger
	engine *xorm.Engine
}

func NewBasicArtifactsStorage(log *Logger, engine *xorm.Engine) (*BasicArtifactsStorage, error) {
	log = log.With(slog.String("entity", "BasicArtifactsStorage"))
	s := &BasicArtifactsStorage{
		log:    log,
		engine: engine,
	}

	return s, nil
}

func (s *BasicArtifactsStorage) NewArtifacts(repo *Repo, artifacts []string, id ArtifactID) (ArtifactID, error) {
	assert(repo != nil)
	assert(len(artifacts) >= 1)
	log := s.log
	storage := repo.Storage
	if id == "" {
		id = ArtifactID(ulid.Make().String())
	}
	log.Info("add artifacts", slog.String("repo", repo.Name), slog.String("storage", storage), slog.Any("files", artifacts), slog.String("id", string(id)))

	if !isDirectoryExist(storage) {
		return "", ErrNoSuchDirectory{storage}
	}

	dest := filepath.Join(storage, string(id))
	if isDirectoryExist(dest) {
		return "", ErrArtifactAlreadyExists{dest}
	}
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return "", err
	}

	input := repo.Input
	for _, fileName := range artifacts {
		// The input mist be sanitized already!!!
		assert(isSecureFileName(fileName))
		// Using input, filename and id detect path withing artifact
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

	return id, nil
}

func (s *BasicArtifactsStorage) Close() {
	log := s.log
	log.Info("closing")
}
