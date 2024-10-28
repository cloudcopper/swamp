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

	repo, err := c.repoRepository.FindByID(repoID, ports.WithRelationship(true))
	if err == ports.ErrRecordNotFound { // 404
		c.renderRepoNotFound(w, repoID, err)
		return
	}
	if err != nil { // 500
		c.renderServerError(w, repoID, err)
		return
	}

	var data *viewmodels.Repo = viewmodels.NewRepo(repo)
	c.render.HTML(w, http.StatusOK, "repo", data)
}

func (c *RepoController) renderRepoNotFound(w http.ResponseWriter, repoID models.RepoID, err error) {
	type Data struct {
		RepoID models.RepoID
		Error  error
	}
	c.render.HTML(w, http.StatusNotFound, "errors/repo-not-found", Data{repoID, err})
}

func (c *RepoController) renderServerError(w http.ResponseWriter, repoID models.RepoID, err error) {
	type Data struct {
		RepoID models.RepoID
		Error  error
	}
	c.render.HTML(w, http.StatusInternalServerError, "errors/repo-server-error", Data{repoID, err})
}
