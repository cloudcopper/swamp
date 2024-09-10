package main

type Error string

func (e Error) Error() string { return string(e) }

const ErrMustBeAbsPath = Error("must be absolute path")
const ErrChecksumFileHasBrokenFiles = Error("checksum file has broken file(s)")
const ErrIsNotChecksumFile = Error("is not checksum file")
