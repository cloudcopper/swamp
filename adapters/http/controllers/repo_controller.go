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

func (c *RepoController) Get(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")

	errors := []string{}
	repo, err := c.repoRepository.FindByID(repoID, ports.WithRelationship(true))
	if err != nil {
		errors = append(errors, err.Error())
	}

	data := struct {
		Errors []string
		Repo   *models.Repo
	}{
		Errors: errors,
		Repo:   repo,
	}

	c.render.HTML(w, http.StatusOK, "repo", data)
}
