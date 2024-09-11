package main

import "fmt"

type Error string

func (e Error) Error() string { return string(e) }

const ErrMustBeAbsPath = Error("must be absolute path")
const ErrChecksumFileHasBrokenFiles = Error("checksum file has broken file(s)")
const ErrIsNotChecksumFile = Error("is not checksum file")
const ErrUnsecureFileName = Error("unsecure file name")

type ErrNoSuchDirectory struct {
	path string
}

func (e ErrNoSuchDirectory) Error() string { return fmt.Sprintf("no such directory %v", e.path) }

type ErrArtifactAlreadyExists struct {
	dest string
}

func (e ErrArtifactAlreadyExists) Error() string {
	return fmt.Sprintf("artifact already exists %v", e.dest)
}
