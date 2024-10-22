package disk

import (
	"io/fs"
	"strings"

	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
)

type FilepathWalk struct {
	fs ports.FS
}

func NewFilepathWalk(f ports.FS) FilepathWalk {
	return FilepathWalk{f}
}

func (f *FilepathWalk) Walk(root string, fn func(name string, err error) (bool, error)) error {
	err := afero.Walk(f.fs, root, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".git") {
			return fs.SkipDir
		}
		ok, err := fn(path, err)
		if !ok {
			return fs.SkipAll
		}
		return err
	})
	return err
}
