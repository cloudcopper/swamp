package infra

import "github.com/unrolled/render"

type Render = *render.Render

func NewRender() Render {
	opts := render.Options{
		Extensions: []string{".tmpl", ".html"},
	}
	r := render.New(opts)
	return r
}
