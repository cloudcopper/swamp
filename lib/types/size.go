package types

import "github.com/dustin/go-humanize"

type Size int64

func (s Size) String() string {
	return humanize.Bytes(uint64(s))
}
