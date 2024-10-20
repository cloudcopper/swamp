package lib

import (
	"errors"
	"os"

	"github.com/spf13/afero"
)

// NoSuchFile return true if file name does not exists
func NoSuchFile(fs afero.Fs, name string) bool {
	if _, err := fs.Stat(name); errors.Is(err, os.ErrNotExist) {
		return true
	}
	return false
}

// FileSize returns size of file or zero
func FileSize(fs afero.Fs, name string) int64 {
	fi, err := fs.Stat(name)
	if err != nil {
		return 0

	}
	return fi.Size()
}

// FileSize2 returns size of file or error
func FileSize2(fs afero.Fs, name string) (int64, error) {
	fi, err := fs.Stat(name)
	if err != nil {
		return 0, err

	}
	return fi.Size(), nil
}

func MoveFile(src afero.Fs, oldname string, dst afero.Fs, newname string) error {
	if src == dst {
		return dst.Rename(oldname, newname)
	}

	panic("move across different FS not yet implemented!!!")
}
