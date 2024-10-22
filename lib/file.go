package lib

import (
	"os"

	"github.com/spf13/afero"
)

// CreateFile creates file name and writes there content.
// The file must not exists.
func CreateFile(fs afero.Fs, name, content string) error {
	f, err := fs.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o660)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
