package controllers

import (
	"log/slog"
	"net/http"

	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/ports"
	"github.com/go-chi/chi/v5"
)

type ArtifactController struct {
	log                ports.Logger
	render             infra.Render
	artifactRepository domain.ArtifactRepository
}

func NewArtifactController(log ports.Logger, render infra.Render, artifactRepository domain.ArtifactRepository) *ArtifactController {
	log = log.With(slog.String("entity", "ArtifactController"))
	s := &ArtifactController{
		log:                log,
		render:             render,
		artifactRepository: artifactRepository,
	}
	return s
}

func (c *ArtifactController) Index(w http.ResponseWriter, r *http.Request) {
	errors := []string{}
	artifacts, err := c.artifactRepository.FindAll()
	if err != nil {
		errors = append(errors, err.Error())
	}

	data := struct {
		Errors    []string
		Artifacts []*models.Artifact
	}{
		Errors:    errors,
		Artifacts: artifacts,
	}

	c.render.HTML(w, http.StatusOK, "artifacts", data)
}

func (c *ArtifactController) Get(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")
	artifactID := chi.URLParam(r, "artifactID")

	errors := []string{}
	artifact, err := c.artifactRepository.FindByID(repoID, artifactID)
	if err != nil {
		errors = append(errors, err.Error())
	}

	data := struct {
		Errors    []string
		Artifacts []*models.Artifact
	}{
		Errors:    errors,
		Artifacts: []*models.Artifact{artifact},
	}

	c.render.HTML(w, http.StatusOK, "artifact", data)
}
