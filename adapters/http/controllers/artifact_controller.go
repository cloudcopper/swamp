package controllers

import (
	"log/slog"
	"net/http"

	"github.com/cloudcopper/swamp/adapters/http/viewmodels"
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
	aritfactStorage    ports.ArtifactStorage
}

func NewArtifactController(log ports.Logger, render infra.Render, artifactRepository domain.ArtifactRepository, aritfactStorage ports.ArtifactStorage) *ArtifactController {
	log = log.With(slog.String("entity", "ArtifactController"))
	s := &ArtifactController{
		log:                log,
		render:             render,
		artifactRepository: artifactRepository,
		aritfactStorage:    aritfactStorage,
	}
	return s
}

func (c *ArtifactController) Index(w http.ResponseWriter, r *http.Request) {
	errors := []string{}
	artifacts, err := c.artifactRepository.FindAll()
	if err != nil {
		errors = append(errors, err.Error())
	}

	perPage := 20
	artifacts, artifactsPage := helperPagination(r, artifacts, perPage)

	data := struct {
		Errors        []string
		Artifacts     []*models.Artifact
		ArtifactsPage int
	}{
		Errors:        errors,
		Artifacts:     artifacts,
		ArtifactsPage: artifactsPage,
	}

	c.render.HTML(w, http.StatusOK, "artifacts", data)
}

func (c *ArtifactController) Get(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")
	artifactID := chi.URLParam(r, "artifactID")

	errors := []string{}

	artifact, err := c.artifactRepository.FindByID(repoID, artifactID, ports.WithRelationship(true))
	// TODO What to do if errors.Is(err, gorm.ErrRecordNotFound)??? 404???
	if err != nil {
		errors = append(errors, err.Error())
	}
	// NOTE The files are not in database atm!!!
	// Should we store those in database?
	// That would be caching and additional validation for tampering?
	files, err := c.aritfactStorage.GetArtifactFiles(artifact.Storage, artifactID)
	if err != nil {
		errors = append(errors, err.Error())
	}
	artifact.Files = files

	data := struct {
		Errors   []string
		Artifact *viewmodels.Artifact
	}{
		Errors:   errors,
		Artifact: viewmodels.NewArtifact(artifact),
	}

	c.render.HTML(w, http.StatusOK, "artifact", data)
}
