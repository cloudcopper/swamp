package infra

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

type Sha256 struct {
}

// Sum return checksum of given file or error
func (s *Sha256) Sum(f ports.FS, fileName string) ([]byte, error) {
	file, err := f.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	// Use hex.EncodeToString to convert to string
	return hash.Sum(nil), nil
}

// CheckFiles return list of good/bad files as specified by checksumFileName
// or first error. Files must be returned with abs path
func (s *Sha256) CheckFiles(f ports.FS, checksumFileName string) (ports.CheckedFiles, error) {
	files := ports.CheckedFiles{}
	dir := filepath.Dir(checksumFileName)

	file, err := f.Open(checksumFileName)
	if err != nil {
		return files, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		a := strings.Fields(line)
		if len(a) != 2 {
			files.Bad = append(files.Bad, line)
			continue
		}
		checksum, fileName := a[0], a[1]
		if !lib.IsSecureFileName(fileName) {
			err = errors.ErrUnsecureFileName
			files.Bad = append(files.Bad, fileName)
			return files, err
		}
		fileName = path.Join(dir, fileName)
		fileName, err = filepath.Abs(fileName)
		if err != nil {
			files.Bad = append(files.Bad, fileName)
			return files, err
		}

		sum, err := s.Sum(f, fileName)
		if err != nil {
			files.Bad = append(files.Bad, fileName)
			return files, err
		}
		if checksum != hex.EncodeToString(sum) {
			files.Bad = append(files.Bad, fileName)
			continue
		}
		files.Good = append(files.Good, fileName)
	}

	if err := scanner.Err(); err != nil {
		return files, err
	}

	return files, err
}

func init() {
	adapters.RegisterChecksumAlgo(100000, "*.sha256sum", &Sha256{})
}
