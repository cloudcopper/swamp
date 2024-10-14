package disk

import (
	"io/fs"
	"strings"

	"github.com/charlievieth/fastwalk"
)

type FilepathWalk struct {
}

func NewFilepathWalk() FilepathWalk {
	return FilepathWalk{}
}

func (f *FilepathWalk) Walk(root string, fn func(name string, err error) (bool, error)) error {
	config := &fastwalk.DefaultConfig
	err := fastwalk.Walk(config, root, func(path string, d fs.DirEntry, err error) error {
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
