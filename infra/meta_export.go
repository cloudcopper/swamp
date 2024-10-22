package infra

import (
	"strings"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
)

type MetaExport struct {
}

func (*MetaExport) ParseMetaFile(f ports.FS, filename string) (map[string]string, error) {
	// Read whole file
	data, err := afero.ReadFile(f, filename)
	if err != nil {
		return nil, err
	}
	s := string(data)

	// Detect prefix
	prefixA, prefixB := "declare -x ", "export "
	a, b := strings.HasPrefix(s, prefixA), strings.HasPrefix(s, prefixB)
	if !a && !b {
		return nil, ports.ErrWrongMetaFormat
	}
	prefix, quote := prefixA, "\""
	if b {
		prefix, quote = prefixB, "'"
	}

	// Parse line by line
	meta := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		// skip empty line
		if line == "\n" || line == "\r" || line == "" {
			continue
		}
		line, found := strings.CutPrefix(line, prefix)
		if !found {
			return nil, ports.ErrWrongMetaFormat
		}
		a := strings.SplitN(line, "=", 2)
		if len(a) != 2 {
			return nil, ports.ErrWrongMetaFormat
		}

		k := a[0]
		v := strings.Trim(a[1], quote)
		meta[k] = v
	}

	return meta, nil
}

func init() {
	adapters.RegisterMetaAlgo(100000, "_export.txt", &MetaExport{})
}
