package infra

import (
	"io/fs"
	"os"

	"github.com/unrolled/render"
)

type Render = *render.Render

func NewRender(f fs.FS, layout string) Render {
	opts := render.Options{
		FileSystem: render.FS(f),
		Extensions: []string{".tmpl", ".html"},
		Layout:     layout,
		IsDevelopment: func() bool {
			return os.Getenv("GO_ENV") == "development"
		}(),
	}
	r := render.New(opts)
	return r
}
