package errors

import (
	"fmt"

	"github.com/cloudcopper/swamp/lib"
)

// TODO Review this file and probably split errors to domain/app specific
//      and implementation/infra specific

const ErrMustBeAbsPath = lib.Error("must be absolute path")
const ErrChecksumFileHasBrokenFiles = lib.Error("checksum file has broken file(s)")
const ErrIsNotChecksumFile = lib.Error("is not checksum file")
const ErrUnsecureFileName = lib.Error("unsecure file name")

type ErrNoSuchDirectory struct {
	Path string
}

func (e ErrNoSuchDirectory) Error() string { return fmt.Sprintf("no such directory %v", e.Path) }

type ErrArtifactAlreadyExists struct {
	Path string
}

func (e ErrArtifactAlreadyExists) Error() string {
	return fmt.Sprintf("artifact already exists %v", e.Path)
}
