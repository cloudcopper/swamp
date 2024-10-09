package infra

import (
	"io/fs"
	"os"

	"github.com/unrolled/render"
)

type Render = *render.Render

func NewRender(fs fs.FS) Render {
	opts := render.Options{
		FileSystem: render.FS(fs),
		Extensions: []string{".tmpl", ".html"},
		IsDevelopment: func() bool {
			return os.Getenv("GO_ENV") == "development"
		}(),
	}
	r := render.New(opts)
	return r
}
