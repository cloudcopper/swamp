package adapters

import (
	"errors"
	"log/slog"
	"path/filepath"
	"sort"

	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

func RegisterMetaAlgo(prio int, pattern string, algo ports.MetaAlgo) {
	info := MetaAlgoInfo{prio, pattern, algo}
	metaAlgos = append(metaAlgos, info)
	sort.Slice(metaAlgos, func(i, j int) bool {
		lib.Assert(metaAlgos[i].prio != metaAlgos[j].prio)
		return metaAlgos[i].prio < metaAlgos[j].prio
	})
}

type MetaAlgoInfo struct {
	prio    int
	pattern string
	algo    ports.MetaAlgo
}

var metaAlgos = []MetaAlgoInfo{}

// IsMetaFile returns true if path match any
// patterns supported by registered algorithms.
// The path must be absolute.
func IsMetaFile(path string) bool {
	lib.Assert(lib.IsAbs(path))
	fileName := filepath.Base(path)

	for _, it := range metaAlgos {
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

// ParseMetaFile parses meta file via algos
// and returns key/values map or error
func ParseMetaFile(log ports.Logger, f ports.FS, metaFileName string) (map[string]string, error) {
	lib.Assert(lib.IsAbs(metaFileName))

	fileName := filepath.Base(metaFileName)
	for _, it := range metaAlgos {
		// Checksum file must match pattern
		if ok, err := filepath.Match(it.pattern, fileName); !ok || err != nil {
			log.Debug("meta filename does not match pattern", slog.String("metaFileName", fileName), slog.String("pattern", it.pattern), slog.Any("err", err))
			continue
		}
		log.Debug("meta filename match pattern", slog.String("metaFileName", fileName), slog.String("pattern", it.pattern))

		// Parse meta file
		meta, err := it.algo.ParseMetaFile(f, metaFileName)
		if errors.Is(err, ports.ErrWrongMetaFormat) {
			continue
		}
		return meta, err
	}

	return nil, nil
}
