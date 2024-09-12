package lib

import (
	"log"
)

// The params may be:
// - error
// - bool, string
func Assert(params ...interface{}) {
	cond := params[0]
	if cond == nil {
		return
	}

	// error
	if e, ok := cond.(error); ok {
		if e != nil {
			log.Fatal(e)
		}
		return
	}

	// bool, string
	if b, ok := cond.(bool); ok {
		if !b {
			msg := params[1].(string)
			log.Fatal(msg)
		}
		return
	}

	panic(cond)
}
