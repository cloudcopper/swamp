package http

import (
	"os"
	"time"

	"github.com/cloudcopper/swamp/ports"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"
)

func NewRouter(log ports.Logger) ports.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(slogchi.New(log))
	r.Use(middleware.Recoverer)
	if os.Getenv("GO_ENV") != "development" {
		r.Use(middleware.Timeout(10 * time.Second))
	}

	return r
}
