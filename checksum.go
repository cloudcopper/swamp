package main

import (
	"encoding/hex"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
)

type ChecksumAlgo interface {
	// Shall return checksum of given file or error
	Sum(fileName string) ([]byte, error)
	// CheckFiles return list of good files, list of bad files as specified by checksumFileName
	// or first error. Files must be returned with abs path
	CheckFiles(checksumFileName string) ([]string, []string, error)
}

func CreateChecksumAlgo(prio int, pattern string, algo ChecksumAlgo) {
	info := ChecksumAlgoInfo{prio, pattern, algo}
	checksumAlgos = append(checksumAlgos, info)
	sort.Slice(checksumAlgos, func(i, j int) bool {
		assert(checksumAlgos[i].prio != checksumAlgos[j].prio)
		return checksumAlgos[i].prio < checksumAlgos[j].prio
	})
}

type ChecksumAlgoInfo struct {
	prio    int
	pattern string
	algo    ChecksumAlgo
}

var checksumAlgos = []ChecksumAlgoInfo{}

func CheckChecksum(log *Logger, checksumFileName string) ([]string, []string, error) {
	assert(filepath.IsAbs(checksumFileName))

	fileName := filepath.Base(checksumFileName)
	for _, it := range checksumAlgos {
		// Checksum file must match pattern
		if ok, err := filepath.Match(it.pattern, fileName); !ok || err != nil {
			log.Debug("checksum filename does not match pattern", slog.String("checksumFileName", fileName), slog.String("pattern", it.pattern), slog.Any("err", err))
			continue
		}
		log.Debug("checksum filename match pattern", slog.String("checksumFileName", fileName), slog.String("pattern", it.pattern))

		// Check checksum file
		checksum, err := it.algo.Sum(checksumFileName)
		if err != nil {
			log.Warn("unable to calc checksum", slog.Any("err", err))
			continue
		}
		expected := strings.Replace(it.pattern, "*", hex.EncodeToString(checksum[:]), 1)
		if expected != fileName {
			log.Warn("checksum file is broken", slog.String("expected", expected), slog.String("checksumFileName", fileName))
			continue
		}
		log.Debug("checksum file is valid", slog.String("expected", expected))

		// Check files listed in valid checksum file
		goodFiles, badFiles, err := it.algo.CheckFiles(checksumFileName)
		switch {
		case err != nil:
			log.Error("unable check content", slog.Any("goodFiles", goodFiles), slog.Any("badFiles", badFiles), slog.Any("err", err))
		case len(badFiles) != 0:
			log.Error("content is partially broken", slog.Any("goodFiles", goodFiles), slog.Any("badFiles", badFiles))
			err = ErrChecksumFileHasBrokenFiles
		default:
			log.Debug("content is fine", slog.Any("goodFiles", goodFiles))
		}
		return goodFiles, badFiles, err
	}

	return nil, nil, ErrIsNotChecksumFile
}
