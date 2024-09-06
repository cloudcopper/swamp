package main

import (
	"log/slog"
	"os"
	"path"
	"testing"

	testifyAssert "github.com/stretchr/testify/assert"
)

func TestWatcherBasic1(t *testing.T) {
	assert := testifyAssert.New(t)

	// prepare test directory
	dir := "testdata/tmp/TestWatcherBasic"
	_ = os.RemoveAll(dir)
	err := os.MkdirAll(dir, os.ModePerm)
	assert.NoError(err)

	// create watcher
	log := slog.Default()
	w, err := NewWatcher(log)
	assert.NoError(err)
	defer w.Close()

	// add dir to watch
	err = w.AddDir(dir)
	assert.NoError(err)

	// create file1
	file1 := path.Join(dir, "file1")
	log.Info("create file", slog.String("file", file1))
	err = createFile(file1, "file 1 line 1\n")
	assert.NoError(err)
	file := <-w.ChanModified
	assert.Equal(file, file1)
	file = <-w.ChanModified
	assert.Equal(file, file1)

	// create file2
	file2 := path.Join(dir, "file2")
	log.Info("create file", slog.String("file", file2))
	err = createFile(file2, "file 2 line 1\n")
	assert.NoError(err)
	file = <-w.ChanModified
	assert.Equal(file, file2)
	file = <-w.ChanModified
	assert.Equal(file, file2)

	// modify file2
	log.Info("append to file", slog.String("file", file2))
	err = appendFile(file2, "file 2 line 2\n")
	assert.NoError(err)
	file = <-w.ChanModified
	assert.Equal(file, file2)

	// move file2 to file3
	file3 := path.Join(dir, "file3")
	log.Info("move file", slog.String("old", file2), slog.String("new", file3))
	err = moveFile(file2, file3)
	assert.NoError(err)
	file = <-w.ChanRemoved
	assert.Equal(file, file2)
	file = <-w.ChanModified
	assert.Equal(file, file3)

	// modify file3
	log.Info("append to file", slog.String("file", file3))
	err = appendFile(file3, "file 3 line 3\n")
	assert.NoError(err)
	file = <-w.ChanModified
	assert.Equal(file, file3)

	// delete file3
	log.Info("delete file", slog.String("file", file3))
	err = deleteFile(file3)
	assert.NoError(err)
	file = <-w.ChanRemoved
	assert.Equal(file, file3)

	// delete file1
	log.Info("delete file", slog.String("file", file1))
	err = deleteFile(file1)
	assert.NoError(err)
	file = <-w.ChanRemoved
	assert.Equal(file, file1)
}

func createFile(name string, content string) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
func appendFile(name string, content string) error {
	f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
func deleteFile(name string) error {
	return os.Remove(name)
}
func moveFile(old, new string) error {
	return os.Rename(old, new)
}
