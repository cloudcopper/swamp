package infra

import (
	"embed"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

//
// Hand crafted basic test of layer fs
//

//go:embed testdata/layer_fs/test1/layer1.d/**
var testLayerFs1 embed.FS

func TestLayerFs1(t *testing.T) {
	assert := require.New(t)
	layer1, err := fs.Sub(testLayerFs1, "testdata/layer_fs/test1/layer1.d")
	assert.NoError(err)

	layerFS, err := NewLayerFileSystem("testdata/layer_fs/test1/layer3.d", "testdata/layer_fs/test1/layer2.d", layer1)
	assert.NoError(err)

	file, err := layerFS.Open(".")
	assert.NoError(err)

	info, err := file.Stat()
	assert.NoError(err)
	assert.True(info.IsDir())

	dir, ok := file.(fs.ReadDirFile)
	assert.True(ok)

	list, err := dir.ReadDir(-1)
	assert.NoError(err)

	files, content := []string{}, []string{}
	for _, l := range list {
		files = append(files, l.Name())

		b, err := layerFS.ReadFile(l.Name())
		assert.NoError(err)
		content = append(content, string(b))
	}

	expFiles := []string{"text0.txt", "text1.txt", "text2.txt", "text3.txt", "text4.txt", "text5.txt"}
	expContent := []string{
		"text0.txt from layer1.d",
		"text1.txt from layer2.d",
		"text2.txt from layer3.d",
		"text3.txt from layer3.d",
		"text4.txt from layer2.d",
		"text5.txt from layer3.d",
	}
	assert.Equal(expFiles, files)
	assert.Equal(expContent, content)
}
