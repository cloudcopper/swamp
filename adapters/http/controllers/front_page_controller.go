package controllers

import (
	"log/slog"
	"net/http"
	"text/template"

	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/ports"
)

type FrontPageController struct {
	log                *ports.Logger
	repoRepository     domain.RepoRepository
	artifactRepository domain.ArtifactRepository
}

func NewFrontPageController(log *ports.Logger,
	repoRepository domain.RepoRepository,
	artifactRepository domain.ArtifactRepository) *FrontPageController {
	log = log.With(slog.String("entity", "FrontPageController"))
	s := &FrontPageController{
		log:                log,
		repoRepository:     repoRepository,
		artifactRepository: artifactRepository,
	}
	return s
}

func (s *FrontPageController) Index(w http.ResponseWriter, r *http.Request) {
	errors := []string{}
	repos, err := s.repoRepository.FindAll()
	if err != nil {
		errors = append(errors, err.Error())
	}
	artifacts, err := s.artifactRepository.FindAll()
	if err != nil {
		errors = append(errors, err.Error())
	}

	data := struct {
		Repos     []*models.Repo
		Artifacts []*models.Artifact
		Errors    []string
	}{
		Repos:     repos,
		Artifacts: artifacts,
		Errors:    errors,
	}

	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, data)
}
