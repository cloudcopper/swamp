package controllers

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

type AboutPageController struct {
	log    ports.Logger
	render infra.Render
}

func NewAboutPageController(log ports.Logger, render infra.Render) *AboutPageController {
	log = log.With(slog.String("entity", "AboutPageController"))
	c := &AboutPageController{
		log:    log,
		render: render,
	}
	return c
}

func (c *AboutPageController) Index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		BuildInfo *debug.BuildInfo
	}{
		BuildInfo: lib.First(debug.ReadBuildInfo()),
	}
	c.render.HTML(w, http.StatusOK, "about", data)
}
