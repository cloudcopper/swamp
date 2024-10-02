package errors

import (
	"fmt"

	"github.com/cloudcopper/swamp/lib"
)

const ErrMustBeAbsPath = lib.Error("must be absolute path")
const ErrChecksumFileHasBrokenFiles = lib.Error("checksum file has broken file(s)")
const ErrIsNotChecksumFile = lib.Error("is not checksum file")
const ErrUnsecureFileName = lib.Error("unsecure file name")
const ErrArtifactIsBroken = lib.Error("artifact is broken")

type ErrArtifactAlreadyExists struct {
	Path string
}

func (e ErrArtifactAlreadyExists) Error() string {
	return fmt.Sprintf("artifact already exists %v", e.Path)
}
