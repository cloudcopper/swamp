package adapters

import (
	"encoding/hex"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

func RegisterChecksumAlgo(prio int, pattern string, algo ports.ChecksumAlgo) {
	info := ChecksumAlgoInfo{prio, pattern, algo}
	checksumAlgos = append(checksumAlgos, info)
	sort.Slice(checksumAlgos, func(i, j int) bool {
		lib.Assert(checksumAlgos[i].prio != checksumAlgos[j].prio)
		return checksumAlgos[i].prio < checksumAlgos[j].prio
	})
}

type ChecksumAlgoInfo struct {
	prio    int
	pattern string
	algo    ports.ChecksumAlgo
}

type ChecksumStr = string

var checksumAlgos = []ChecksumAlgoInfo{}

// IsChecksumFile returns true if path match any
// patterns supported by registered algorithms.
// The path must be absolute.
func IsChecksumFile(path string) bool {
	lib.Assert(lib.IsAbs(path))
	fileName := filepath.Base(path)

	for _, it := range checksumAlgos {
		ok, err := filepath.Match(it.pattern, fileName)
		if err != nil {
			continue
		}
		if ok {
			return true
		}
	}

	return false
}

// CheckChecksum checks the checksumFileName
// and all files listed inside the checksumFileName.
// It returns the checksum of the checksumFileName,
// good files, broken files and error
func CheckChecksum(log ports.Logger, f ports.FS, checksumFileName string) (ChecksumStr, ports.CheckedFiles, error) {
	lib.Assert(lib.IsAbs(checksumFileName))

	fileName := filepath.Base(checksumFileName)
	for _, it := range checksumAlgos {
		// Checksum file must match pattern
		if ok, err := filepath.Match(it.pattern, fileName); !ok || err != nil {
			log.Debug("checksum filename does not match pattern", slog.String("checksumFileName", fileName), slog.String("pattern", it.pattern), slog.Any("err", err))
			continue
		}
		log.Debug("checksum filename match pattern", slog.String("checksumFileName", fileName), slog.String("pattern", it.pattern))

		// Check checksum file
		checksum, err := it.algo.Sum(f, checksumFileName)
		if err != nil {
			log.Warn("unable to calc checksum", slog.Any("err", err))
			continue
		}
		expected := strings.Replace(it.pattern, "*", hex.EncodeToString(checksum), 1)
		if expected != fileName {
			log.Warn("checksum file is broken", slog.String("expected", expected), slog.String("checksumFileName", fileName))
			continue
		}
		log.Debug("checksum file is valid", slog.String("expected", expected))

		// Check files listed in valid checksum file
		files, err := it.algo.CheckFiles(f, checksumFileName)
		switch {
		case err != nil:
			log.Error("unable check content", slog.Any("files.Good", files.Good), slog.Any("files.Bad", files.Bad), slog.Any("err", err))
		case len(files.Bad) != 0:
			log.Error("content is partially broken", slog.Any("files.Good", files.Good), slog.Any("files.Bad", files.Bad))
			err = errors.ErrChecksumFileHasBrokenFiles
		default:
			log.Debug("content is fine", slog.Any("files.Good", files.Good))
		}
		return hex.EncodeToString(checksum), files, err
	}

	return "", ports.CheckedFiles{}, errors.ErrIsNotChecksumFile
}
