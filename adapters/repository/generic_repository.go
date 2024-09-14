package repository

import (
	"fmt"

	"xorm.io/xorm"
)

func findAll[T any](engine *xorm.Engine) ([]*T, error) {
	var v []*T
	err := engine.Find(&v)
	return v, err
}

// The iterateAll iterates until callback return false or error
func iterateAll[T any](engine *xorm.Engine, callback func(repo *T) (bool, error)) error {
	errBreak := fmt.Errorf("break - not an error")
	err := engine.Iterate(new(T), func(_ int, bean interface{}) error {
		rec := bean.(*T)
		ok, err := callback(rec)
		if !ok {
			return errBreak
		}
		return err
	})
	if err == errBreak {
		return nil
	}
	return err
}
