package infra

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cloudcopper/swamp/lib"
)

type FileSystem interface {
	fs.ReadFileFS
	fs.ReadDirFS
}

type LayerFileSystem struct {
	layers []FileSystem
}

const ErrWrongParamType = lib.Error("wrong param type")
const ErrNotReadDirFileFS = lib.Error("is not interface to fs.ReadDirFileFS")

func NewLayerFileSystem(params ...interface{}) (*LayerFileSystem, error) {
	l := &LayerFileSystem{}
	for _, p := range params {
		switch v := p.(type) {
		case func() (string, error):
			path, err := v()
			if err != nil {
				return nil, err
			}
			if path == "" {
				continue
			}
			lib.Assert(!strings.Contains(path, ".."))
			l.layers = append(l.layers, &osFileSystem{path})

		case string:
			v = strings.TrimSpace(v)
			if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
				name := v[2 : len(v)-1]
				path := os.Getenv(name)
				if path == "" {
					continue
				}
				v = path
			}
			lib.Assert(!strings.Contains(v, ".."))
			l.layers = append(l.layers, &osFileSystem{v})

		case embed.FS:
			l.layers = append(l.layers, &embedFileSystem{v})

		case fs.SubFS:
			l.layers = append(l.layers, &subFileSystem{v})

		default:
			return nil, ErrWrongParamType
		}

	}

	return l, nil
}

func (l *LayerFileSystem) Open(name string) (fs.File, error) {
	lib.Assert(!strings.Contains(name, ".."))
	for _, layer := range l.layers {
		f, err := layer.Open(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}

		info, err := f.Stat()
		if err != nil && !info.IsDir() {
			return f, err
		}
		d := &layerDir{f, l, name}
		return d, nil
	}
	return nil, os.ErrNotExist
}
func (l *LayerFileSystem) ReadFile(name string) ([]byte, error) {
	lib.Assert(!strings.Contains(name, ".."))
	for _, layer := range l.layers {
		data, err := layer.ReadFile(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return data, err
	}
	return nil, os.ErrNotExist
}

type embedFileSystem struct {
	embed.FS
}

type subFileSystem struct {
	fs.SubFS
}

func (s *subFileSystem) ReadFile(name string) ([]byte, error) {
	file, err := s.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fs.ErrInvalid
	}
	size := info.Size()
	buf := make([]byte, size)
	n, err := file.Read(buf)
	lib.Assert(n > 0)
	buf = buf[:n]
	return buf, err
}

func (s *subFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := s.Open(".")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fs.ErrInvalid
	}

	dir, ok := file.(fs.ReadDirFS)
	if !ok {
		return nil, ErrNotReadDirFileFS
	}

	return dir.ReadDir(name)
}

type osFileSystem struct {
	root string
}

func (o *osFileSystem) Open(name string) (fs.File, error) {
	path := filepath.Join(o.root, name)
	return os.Open(path)
}

func (o *osFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	path := filepath.Join(o.root, name)
	return os.ReadDir(path)
}

func (o *osFileSystem) ReadFile(name string) ([]byte, error) {
	path := filepath.Join(o.root, name)
	return os.ReadFile(path)
}

type layerDir struct {
	fs.File
	fs   *LayerFileSystem
	name string
}

func (l *layerDir) ReadDir(n int) ([]fs.DirEntry, error) {
	m := make(map[string]fs.DirEntry)

	for _, layer := range l.fs.layers {
		file, err := layer.Open(l.name)
		if err != nil {
			continue
		}
		info, err := file.Stat()
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}

		dir, ok := file.(fs.ReadDirFile)
		if !ok {
			continue
		}
		entries, err := dir.ReadDir(-1)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			name := e.Name()
			if _, exists := m[name]; exists {
				continue
			}
			m[name] = e
		}
	}

	entries := []fs.DirEntry{}
	for _, e := range m {
		entries = append(entries, e)
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		if a.Name() < b.Name() {
			return -1
		}
		if a.Name() > b.Name() {
			return 1
		}
		return 0
	})

	return entries, nil
}
