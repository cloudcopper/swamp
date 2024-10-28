package controllers

import (
	"log/slog"
	"net/http"

	"github.com/cloudcopper/swamp/adapters/http/viewmodels"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

type FrontPageController struct {
	log    ports.Logger
	render infra.Render
	repos  domain.Repositories
}

func NewFrontPageController(log ports.Logger, render infra.Render, repos domain.Repositories) *FrontPageController {
	log = log.With(slog.String("entity", "FrontPageController"))
	c := &FrontPageController{
		log:    log,
		render: render,
		repos:  repos,
	}
	return c
}

func (c *FrontPageController) Index(w http.ResponseWriter, r *http.Request) {
	errors := []string{}
	repos, err := c.repos.Repo().FindAll(ports.WithRelationship(true), ports.LimitArtifacts(1))
	if len(repos) == 0 && err == nil {
		err = lib.Error("ERROR: No repository found!!! Configuration problem ???")
	}
	if err != nil {
		errors = append(errors, err.Error())
	}

	artifacts, err := c.repos.Artifact().FindAll()
	if err != nil {
		errors = append(errors, err.Error())
	}

	perPage := 20
	artifacts, artifactsPage := helperPagination(r, artifacts, perPage)

	data := struct {
		Errors        []string
		Repos         []*viewmodels.Repo
		Artifacts     []*viewmodels.Artifact
		ArtifactsPage int
	}{
		Errors:        errors,
		Repos:         viewmodels.NewRepos(repos),
		Artifacts:     viewmodels.NewArtifacts(artifacts),
		ArtifactsPage: artifactsPage,
	}

	c.render.HTML(w, http.StatusOK, "index", data)
}

// NotFound is a custom 404 handler
func (c *FrontPageController) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	c.render.HTML(w, http.StatusNotFound, "errors/404", nil)
}
