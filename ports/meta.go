package ports

import "github.com/cloudcopper/swamp/lib"

const ErrWrongMetaFormat = lib.Error("wrong meta format")

type MetaAlgo interface {
	ParseMetaFile(fs FS, metaFileName string) (map[string]string, error)
}
