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
	"github.com/spf13/afero"
)

type BasicArtifactStorageAdapter struct {
	log ports.Logger
	fs  ports.FS
}

func NewBasicArtifactStorageAdapter(log ports.Logger, f ports.FS) (*BasicArtifactStorageAdapter, error) {
	log = log.With(slog.String("entity", "BasicArtifactStorageAdapter"))
	s := &BasicArtifactStorageAdapter{
		log: log,
		fs:  f,
	}

	return s, nil
}

func (s *BasicArtifactStorageAdapter) NewArtifact(src ports.FS, input string, artifacts []string, storage string, id models.ArtifactID) (*ports.NewArtifactInfo, error) {
	lib.Assert(storage != "")
	lib.Assert(id != "")
	lib.Assert(len(artifacts) >= 1)
	log, dst := s.log, s.fs
	log = log.With(slog.Any("storage", storage), slog.String("artifactID", string(id)))
	log.Info("add artifacts", slog.Any("input", input), slog.Any("files", artifacts))

	exist, _ := afero.DirExists(dst, storage)
	if !exist {
		return nil, lib.ErrNoSuchDirectory{Path: storage}
	}

	dest := filepath.Join(storage, string(id))
	exist, _ = afero.DirExists(dst, dest)
	if exist {
		return nil, errors.ErrArtifactAlreadyExists{Path: dest}
	}
	if err := dst.MkdirAll(dest, os.ModePerm); err != nil {
		return nil, err
	}

	// Move all artifacts
	size := int64(0)
	for _, fileName := range artifacts {
		// The input must be sanitized already!!!
		lib.Assert(lib.IsSecureFileName(fileName))
		lib.Assert(strings.HasPrefix(fileName, input))

		// Using input, fileName and id to detect path withing artifact
		name := fileName
		name = strings.TrimPrefix(name, input)
		name = strings.TrimPrefix(name, string(os.PathSeparator))
		name = strings.TrimPrefix(name, id+string(os.PathSeparator))
		dir, file := filepath.Split(name)
		dest := filepath.Join(dest, dir)
		if dir != "" {
			if err := dst.MkdirAll(dest, os.ModePerm); err != nil {
				return nil, err
			}
		}
		newpath := filepath.Join(dest, file)
		// Move single artifact
		if err := lib.MoveFile(src, fileName, dst, newpath); err != nil {
			return nil, err
		}
		size += lib.FileSize(dst, newpath)
	}

	// Optional create file _createdAt.txt containing epoch time.
	// It can be part of artifacts as well.
	// In such case the creation time would be preserved by checksum file.
	// Can be created by ```date +%s > _createdAt.txt```
	now := time.Now().UTC().Unix()
	file := filepath.Join(dest, "_createdAt.txt")
	if err := lib.CreateFile(dst, file, fmt.Sprintf("%v", now)); lib.NoSuchFile(dst, file) && err != nil {
		log.Warn("unable to create", slog.String("file", file), slog.Any("err", err))
	}

	// Read back creation time
	a, err := afero.ReadFile(dst, file)
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

	info := &ports.NewArtifactInfo{
		Size:      size,
		CreatedAt: createdAt,
	}

	return info, nil
}

func (s *BasicArtifactStorageAdapter) RemoveArtifact(storage string, artifactID models.ArtifactID) error {
	path := filepath.Join(storage, artifactID)
	err := s.fs.RemoveAll(path)
	return err
}

func (s *BasicArtifactStorageAdapter) OpenFile(storage string, artifactID models.ArtifactID, filename string) (ports.File, error) {
	path := filepath.Join(storage, artifactID, filename)
	f, err := s.fs.OpenFile(path, os.O_RDONLY, 0)
	return f, err
}

func (s *BasicArtifactStorageAdapter) Close() {
	log := s.log
	log.Info("closing")
}
