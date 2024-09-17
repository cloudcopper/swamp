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

type RepoController struct {
	log            ports.Logger
	render         infra.Render
	repoRepository domain.RepoRepository
}

func NewRepoController(log ports.Logger, render infra.Render, repoRepository domain.RepoRepository) *RepoController {
	log = log.With(slog.String("entity", "RepoController"))
	s := &RepoController{
		log:            log,
		render:         render,
		repoRepository: repoRepository,
	}
	return s
}

func (c *RepoController) Index(w http.ResponseWriter, r *http.Request) {
	errors := []string{}
	repos, err := c.repoRepository.FindAllWithRelations()
	if err != nil {
		errors = append(errors, err.Error())
	}

	data := struct {
		Errors []string
		Repos  []*models.Repo
	}{
		Errors: errors,
		Repos:  repos,
	}

	c.render.HTML(w, http.StatusOK, "repos", data)
}

func (c *RepoController) Get(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")

	errors := []string{}
	repos, err := c.repoRepository.FindAllByID(repoID)
	if err != nil {
		errors = append(errors, err.Error())
	}

	data := struct {
		Errors []string
		Repos  []*models.Repo
	}{
		Errors: errors,
		Repos:  repos,
	}

	c.render.HTML(w, http.StatusOK, "repo", data)
}
