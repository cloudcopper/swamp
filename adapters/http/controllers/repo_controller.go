package controllers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/ports"
)

type RepoController struct {
	log            *ports.Logger
	repoRepository domain.RepoRepository
}

func NewRepoController(log *ports.Logger, repoRepository domain.RepoRepository) *RepoController {
	log = log.With(slog.String("entity", "RepoController"))
	s := &RepoController{
		log:            log,
		repoRepository: repoRepository,
	}
	return s
}

func (s *RepoController) Index(w http.ResponseWriter, r *http.Request) {
	repos, err := s.repoRepository.FindAll()
	if err != nil {
		fmt.Fprintf(w, "RepoController.Index fetching all repos error - %v</br>", err)
	}
	for i, repo := range repos {
		fmt.Fprintf(w, "Repo %v:</br>%v</br>", i, repo)
	}
}

func (s *RepoController) Get(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "RepoController.Get is called!!!")
}
