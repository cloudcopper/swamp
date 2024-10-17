package adapters

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/infra/disk"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/lib/types"
	"github.com/cloudcopper/swamp/ports"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/afero"
)

type BasicArtifactStorageAdapter struct {
	log ports.Logger
	fs  ports.FS
}

func NewBasicArtifactStorageAdapter(log ports.Logger, fs ports.FS) (*BasicArtifactStorageAdapter, error) {
	log = log.With(slog.String("entity", "BasicArtifactStorageAdapter"))
	s := &BasicArtifactStorageAdapter{
		log: log,
		fs:  fs,
	}

	return s, nil
}

func (s *BasicArtifactStorageAdapter) NewArtifact(input string, storage string, id models.ArtifactID, artifacts []string) (*ports.NewArtifactInfo, error) {
	lib.Assert(storage != "")
	lib.Assert(len(artifacts) >= 1)
	log, fs := s.log, s.fs
	if id == "" {
		id = ulid.Make().String()
	}
	log = log.With(slog.Any("storage", storage), slog.String("artifactID", string(id)))
	log.Info("add artifacts", slog.Any("input", input), slog.Any("files", artifacts))

	exist, _ := afero.DirExists(fs, storage)
	if !exist {
		return nil, lib.ErrNoSuchDirectory{Path: storage}
	}

	dest := filepath.Join(storage, string(id))
	exist, _ = afero.DirExists(fs, dest)
	if exist {
		return nil, errors.ErrArtifactAlreadyExists{Path: dest}
	}
	if err := fs.MkdirAll(dest, os.ModePerm); err != nil {
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
			if err := fs.MkdirAll(dest, os.ModePerm); err != nil {
				return nil, err
			}
		}
		newpath := filepath.Join(dest, file)
		// Move single artifact
		if err := fs.Rename(fileName, newpath); err != nil {
			return nil, err
		}
		size = size + lib.FileSize(fs, newpath)
	}

	// Optional create file _createdAt.txt containing epoch time.
	// It can be part of artifacts as well.
	// In such case the creation time would be preserved by checksum file.
	// Can be created by ```date +%s > _createdAt.txt```
	now := time.Now().UTC().Unix()
	file := filepath.Join(dest, "_createdAt.txt")
	if err := lib.CreateFile(fs, file, fmt.Sprintf("%v", now)); lib.NoSuchFile(fs, file) && err != nil {
		log.Warn("unable to create", slog.String("file", file), slog.Any("err", err))
	}

	// Read back creation time
	a, err := afero.ReadFile(fs, file)
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
		ID:        id,
		Size:      size,
		CreatedAt: createdAt,
	}

	return info, nil
}

func (s *BasicArtifactStorageAdapter) GetArtifactFiles(storage string, artifactID models.ArtifactID) (models.ArtifactFiles, error) {
	log, fs := s.log, s.fs

	exist, _ := afero.DirExists(fs, storage)
	if !exist {
		log.Error("storage not found", slog.String("storage", storage))
		return nil, lib.ErrNoSuchDirectory{Path: storage}
	}
	path := filepath.Join(storage, artifactID)
	exist, _ = afero.DirExists(fs, path)
	if !exist {
		log.Error("artifact not found", slog.String("path", path))
		return nil, lib.ErrNoSuchDirectory{Path: path}
	}

	files := models.ArtifactFiles{}
	w := disk.NewFilepathWalk(fs)
	w.Walk(path, func(name string, err error) (bool, error) {
		if err != nil {
			log.Error("filepath walk failed", slog.String("path", path), slog.Any("err", err))
			return true, err
		}
		lib.Assert(strings.HasPrefix(name, path))
		if name == path {
			return true, nil
		}
		files = append(files, &models.ArtifactFile{
			Name:  name,
			Size:  types.Size(lib.FileSize(fs, name)),
			State: vo.ArtifactIsOK, // TODO Ideally we would have to check the checksum somehow here
		})
		return true, nil
	})

	slices.SortFunc(files, func(a, b *models.ArtifactFile) int {
		s := []string{
			strings.TrimPrefix(a.Name, path+string(filepath.Separator)),
			strings.TrimPrefix(b.Name, path+string(filepath.Separator)),
		}
		for i := range s {
			if strings.HasPrefix(s[i], "_created") {
				s[i] = "zzzz" + s[i]
			} else if s[i][0] == '_' {
				s[i] = "zzz" + s[i]
			}
			if strings.HasSuffix(s[i], ".md5") {
				s[i] = "zzzzzz" + s[i]
			}
			if strings.HasSuffix(s[i], ".sha256sum") {
				s[i] = "zzzzzzz" + s[i]
			}
		}

		if s[0] > s[1] {
			return 1
		}
		if s[0] < s[1] {
			return -1
		}
		return 0
	})
	return files, nil
}

func (s *BasicArtifactStorageAdapter) Close() {
	log := s.log
	log.Info("closing")
}
