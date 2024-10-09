package infra

import (
	"database/sql"

	"github.com/cloudcopper/swamp/ports"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // purego sqlite3 driver
)

const (
	DriverSqlite         = "sqlite"
	SourceSqliteInMemory = "file::memory:?cache=shared&_pragma=foreign_keys(1)"
)

func NewDatabase(log ports.Logger, driver, source string) (ports.DB, func(), error) {
	sqlDB, err := sql.Open(driver, source)
	if err != nil {
		return nil, nil, err
	}

	dbLogger := slogGorm.New(slogGorm.WithHandler(log.Handler()))
	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		sqlDB.Close()
		return nil, nil, err
	}

	return db, func() { sqlDB.Close() }, nil
}
