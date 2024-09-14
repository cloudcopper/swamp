package controllers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/ports"
)

type ArtifactController struct {
	log                *ports.Logger
	artifactRepository domain.ArtifactRepository
}

func NewArtifactController(log *ports.Logger, artifactRepository domain.ArtifactRepository) *ArtifactController {
	log = log.With(slog.String("entity", "ArtifactController"))
	s := &ArtifactController{
		log:                log,
		artifactRepository: artifactRepository,
	}
	return s
}

func (s *ArtifactController) Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ArtifactController.Index is called!!!")
}

func (s *ArtifactController) Get(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ArtifactController.Get is called!!!")
}
