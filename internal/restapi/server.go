package restapi

import (
	"log/slog"
	"net/http"

	"github.com/AlejandroHerr/go-common/pkg/api"
	"github.com/AlejandroHerr/go-idasen-desk/internal/idasen"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
)

type Config struct {
	Port uint
}

func NewHandler(
	authTokens []string,
	manager *idasen.Manager,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)

	r.Use(api.RequestIDMiddleware())
	r.Use(api.RequestLoggerMiddleware(logger))
	r.Use(middleware.URLFormat)
	r.Use(middleware.NoCache)
	r.Use(cors.Handler(cors.Options{ //nolint:exhaustruct // idk
		AllowedOrigins: []string{"*"},
	}))
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/status", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := map[string]string{"status": "ok"}
		render.JSON(w, r, ok)
	}))

	v1router := NewV1Router(authTokens, manager, logger)

	r.Mount("/v1", v1router)

	return r
}
