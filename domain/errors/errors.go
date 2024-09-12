package errors

import "fmt"

type Error string

func (e Error) Error() string { return string(e) }

const ErrMustBeAbsPath = Error("must be absolute path")
const ErrChecksumFileHasBrokenFiles = Error("checksum file has broken file(s)")
const ErrIsNotChecksumFile = Error("is not checksum file")
const ErrUnsecureFileName = Error("unsecure file name")

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
