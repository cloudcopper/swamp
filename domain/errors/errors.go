package errors

import (
	"errors"
	"fmt"

	"github.com/cloudcopper/swamp/lib"
)

const ErrMustBeAbsPath = lib.Error("must be absolute path")
const ErrChecksumFileHasBrokenFiles = lib.Error("checksum file has broken file(s)")
const ErrIsNotChecksumFile = lib.Error("is not checksum file")
const ErrUnsecureFileName = lib.Error("unsecure file name")
const ErrArtifactIsBroken = lib.Error("artifact is broken")
const ErrIncorrectMetaID = lib.Error("incorrect meta id")
const ErrIncorrectFileID = lib.Error("incorrect file id")
const ErrNotMatchRepoInput = lib.Error("not match repo input")

type ErrArtifactAlreadyExists struct {
	Path string
}

func (e ErrArtifactAlreadyExists) Error() string {
	return fmt.Sprintf("artifact already exists %v", e.Path)
}

var Is = errors.Is
