package adapters

import (
	"github.com/cloudcopper/swamp/ports"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() ports.Router {
	router := chi.NewRouter()
	router.Use(middleware.Logger) // TODO Replace with slog-chi
	router.Use(middleware.Recoverer)

	return router
}
