package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Sha256 struct {
}

// Sum return checksum of given file or error
func (s *Sha256) Sum(fileName string) ([]byte, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	// hex.EncodeToString(
	return hash.Sum(nil), nil
}

// CheckFiles return list of good files, list of bad files as specified by checksumFileName
// or first error. Files must be returned with abs path
func (s *Sha256) CheckFiles(checksumFileName string) ([]string, []string, error) {
	goodFiles, badFiles := []string{}, []string{}
	dir := filepath.Dir(checksumFileName)

	file, err := os.Open(checksumFileName)
	if err != nil {
		return goodFiles, badFiles, err
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
			badFiles = append(badFiles, line)
			continue
		}
		checksum, fileName := a[0], a[1]
		fileName = path.Join(dir, fileName)
		fileName, err = filepath.Abs(fileName)
		if err != nil {
			badFiles = append(badFiles, fileName)
			return goodFiles, badFiles, err
		}

		sum, err := s.Sum(fileName)
		if err != nil {
			badFiles = append(badFiles, fileName)
			return goodFiles, badFiles, err
		}
		if checksum != hex.EncodeToString(sum) {
			badFiles = append(badFiles, fileName)
			continue
		}
		goodFiles = append(goodFiles, fileName)
	}

	if err := scanner.Err(); err != nil {
		return goodFiles, badFiles, err
	}

	return goodFiles, badFiles, err
}

func init() {
	CreateChecksumAlgo(100000, "*.sha256sum", &Sha256{})
}
