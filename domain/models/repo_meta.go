package models

import (
	"github.com/cloudcopper/swamp/lib"
	"gopkg.in/yaml.v3"
)

type RepoMeta struct {
	RepoID RepoID `gorm:"primaryKey;not null" validate:"required,validid"`
	Key    string `gorm:"primaryKey;not null" validate:"required"`
	Value  string
}

type RepoMetas []*RepoMeta

func (meta *RepoMetas) UnmarshalYAML(value *yaml.Node) error {
	// Create a temporary map to hold the key-value pairs
	var m map[string]string
	if err := value.Decode(&m); err != nil {
		return err
	}

	// Convert the map into a slice of KeyValue
	for k, v := range m {
		*meta = append(*meta, &RepoMeta{Key: k, Value: v})
	}

	return nil
}

func (metas *RepoMetas) Secure() {
	a := RepoMetas{}
	for _, m := range *metas {
		if lib.IsKeyBlacklisted(m.Key) {
			continue
		}
		if lib.IsKeyValueBlacklisted(m.Key) {
			m.Value = "********************************"
		}
		a = append(a, m)
	}
	*metas = a
}
