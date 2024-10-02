package infra

import (
	"io/fs"

	"github.com/unrolled/render"
)

type Render = *render.Render

func NewRender(fs fs.FS) Render {
	opts := render.Options{
		FileSystem: render.FS(fs),
		Extensions: []string{".tmpl", ".html"},
	}
	r := render.New(opts)
	return r
}
