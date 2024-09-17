package repository

import (
	"github.com/cloudcopper/swamp/ports"
)

// The iterateAll iterates until callback return false or error
func iterateAll[T any](db ports.DB, callback func(repo *T) (bool, error)) error {
	rows, err := db.Model(new(T)).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var model T
		if err := db.ScanRows(rows, &model); err != nil {
			return err
		}

		ok, err := callback(&model)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
	}

	return nil
}

/*

	for rows.Next() {
		var repo models.Repo
		if err := r.db.ScanRows(rows, &repo); err != nil {
			return err
		}

		ok, err := callback(&repo)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
	}

	return nil
*/
