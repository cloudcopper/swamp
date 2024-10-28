package infra

import (
	"html/template"
	"io/fs"
	"os"
	"reflect"
	"strings"

	"github.com/unrolled/render"
)

type Render = *render.Render

func NewRender(f fs.FS, layout string) Render {
	opts := render.Options{
		FileSystem: render.FS(f),
		Extensions: []string{".tmpl", ".html"},
		Layout:     layout,
		Funcs: []template.FuncMap{
			{
				"hasField":  hasField,
				"hasPrefix": hasPrefix,
				"isHref":    isHref,
			},
		},
		IsDevelopment: func() bool {
			return os.Getenv("GO_ENV") == "development"
		}(),
	}
	r := render.New(opts)
	return r
}

func hasField(data interface{}, fieldName string) bool {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		return v.FieldByName(fieldName).IsValid()
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func isHref(s string) bool {
	hrefs := []string{
		"https://",
		"http://",
		"mailto:",
	}

	for _, href := range hrefs {
		if strings.HasPrefix(s, href) {
			return true
		}
	}

	return false
}
