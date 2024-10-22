package ports

type CheckedFiles struct {
	Good []string
	Bad  []string
}
type ChecksumAlgo interface {
	// Shall return checksum of given file or error
	Sum(fs FS, fileName string) ([]byte, error)
	// CheckFiles return list of good files, list of bad files as specified by checksumFileName
	// or first error. Files must be returned with abs path
	CheckFiles(fs FS, checksumFileName string) (CheckedFiles, error)
}
