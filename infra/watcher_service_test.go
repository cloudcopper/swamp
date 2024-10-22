package infra

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/cloudcopper/swamp/lib"
	"github.com/spf13/afero"
	testifyAssert "github.com/stretchr/testify/assert"
)

func TestWatcherServiceBasic1(t *testing.T) {
	fs := afero.NewOsFs()
	assert := testifyAssert.New(t)

	// Prepare test directory
	// NOTE The watcher service needs abs path, so operate with abs path from beginning
	dir, err := filepath.Abs("testdata/tmp/TestWatcherServiceBasic1")
	assert.NoError(err)
	if err != nil {
		t.FailNow()
	}
	_ = os.RemoveAll(dir) // ignore result as the test directory may not exists
	err = os.MkdirAll(dir, os.ModePerm)
	assert.NoError(err)
	if err != nil {
		t.FailNow()
	}

	// Create watcher service
	log := slog.Default()
	bus := NewEventBus()
	defer bus.Shutdown()
	s, err := NewWatcherService("TestWatcherServiceBasic1", log, bus)
	assert.NoError(err)
	defer s.Close()

	chanModified := bus.Sub(fmt.Sprintf("%v-file-modified", s.id))
	chanRemoved := bus.Sub(fmt.Sprintf("%v-file-removed", s.id))

	// Add dir to watch
	err = s.addDir(dir)
	assert.NoError(err)
	if err != nil {
		t.FailNow()
	}

	// Create file1
	file1 := path.Join(dir, "file1")
	log.Info("create file", slog.String("file", file1))
	err = lib.CreateFile(fs, file1, "file 1 line 1\n")
	assert.NoError(err)
	file := <-chanModified
	assert.Equal(file[0], file1)
	file = <-chanModified
	assert.Equal(file[0], file1)

	// Create file2
	file2 := path.Join(dir, "file2")
	log.Info("create file", slog.String("file", file2))
	err = lib.CreateFile(fs, file2, "file 2 line 1\n")
	assert.NoError(err)
	file = <-chanModified
	assert.Equal(file[0], file2)
	file = <-chanModified
	assert.Equal(file[0], file2)

	// Modify file2
	log.Info("append to file", slog.String("file", file2))
	err = appendFile(file2, "file 2 line 2\n")
	assert.NoError(err)
	file = <-chanModified
	assert.Equal(file[0], file2)

	// Move file2 to file3
	file3 := path.Join(dir, "file3")
	log.Info("move file", slog.String("old", file2), slog.String("new", file3))
	err = os.Rename(file2, file3)
	assert.NoError(err)
	file = <-chanRemoved
	assert.Equal(file[0], file2)
	file = <-chanModified
	assert.Equal(file[0], file3)

	// Modify file3
	log.Info("append to file", slog.String("file", file3))
	err = appendFile(file3, "file 3 line 3\n")
	assert.NoError(err)
	file = <-chanModified
	assert.Equal(file[0], file3)

	// Delete file3
	log.Info("delete file", slog.String("file", file3))
	err = os.Remove(file3)
	assert.NoError(err)
	file = <-chanRemoved
	assert.Equal(file[0], file3)

	// Delete file1
	log.Info("delete file", slog.String("file", file1))
	err = os.Remove(file1)
	assert.NoError(err)
	file = <-chanRemoved
	assert.Equal(file[0], file1)
}

func appendFile(name, content string) error {
	f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0o660)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
