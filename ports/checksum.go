package ports

type ChecksumAlgo interface {
	// Shall return checksum of given file or error
	Sum(fileName string) ([]byte, error)
	// CheckFiles return list of good files, list of bad files as specified by checksumFileName
	// or first error. Files must be returned with abs path
	CheckFiles(checksumFileName string) ([]string, []string, error)
}
