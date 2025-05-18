package restapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/AlejandroHerr/go-common/pkg/api"
	"github.com/AlejandroHerr/go-idasen-desk/internal/idasen"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

func NewV1Router(manager *idasen.Manager, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Get("/desk/{id}", api.HandleRendererFunc(
		func(_ http.ResponseWriter, r *http.Request) (render.Renderer, *api.ErrRepsonse) {
			id := chi.URLParam(r, "id")
			if _, err := uuid.Parse(id); err != nil {
				return nil, api.NewErrorResponse(
					err,
					http.StatusBadRequest,
					http.StatusText(http.StatusBadRequest),
					"Invalid UUID",
					nil,
				)
			}

			height, err := manager.ReadHeight(id)
			if err != nil {
				logger.ErrorContext(r.Context(), "Error reading height", slog.String("error", err.Error()))

				return nil, api.NewErrorResponse(
					err,
					http.StatusInternalServerError,
					http.StatusText(http.StatusInternalServerError),
					"Failed to read height",
					nil,
				)
			}

			return NewHeightResponse(height), nil
		},
		logger,
	))

	r.Patch("/desk/{id}", api.HandleRendererFunc(
		func(_ http.ResponseWriter, r *http.Request) (render.Renderer, *api.ErrRepsonse) {
			id := chi.URLParam(r, "id")
			if _, err := uuid.Parse(id); err != nil {
				return nil, api.NewErrorResponse(
					err,
					http.StatusBadRequest,
					http.StatusText(http.StatusBadRequest),
					"Invalid UUID",
					nil,
				)
			}

			var req MoveToRquest
			if err := render.Bind(r, &req); err != nil {
				return nil, api.NewErrorResponse(
					err,
					http.StatusBadRequest,
					http.StatusText(http.StatusBadRequest),
					"Invalid request",
					nil,
				)
			}

			height, err := manager.MoveTo(r.Context(), id, req.Height)
			if err != nil {
				logger.ErrorContext(r.Context(), "Error moving to height", slog.String("error", err.Error()))

				return nil, api.NewErrorResponse(
					err,
					http.StatusInternalServerError,
					http.StatusText(http.StatusInternalServerError),
					"Failed to move to height",
					nil,
				)
			}

			return NewHeightResponse(height), nil
		},
		logger,
	))

	return r
}

type MoveToRquest struct {
	Height int `json:"height"`
}

var _ render.Binder = (*MoveToRquest)(nil)

func (m *MoveToRquest) Bind(_ *http.Request) error {
	if m.Height < 0 {
		return errors.New("height must be greater than or equal to 0")
	}

	return nil
}

type HeightResponse struct {
	Height int `json:"height"`
}

var _ render.Renderer = (*HeightResponse)(nil)

func NewHeightResponse(height int) *HeightResponse {
	return &HeightResponse{
		Height: height,
	}
}

func (h *HeightResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	render.Status(r, http.StatusOK)

	return nil
}

type OkResponse struct {
}

var _ render.Renderer = (*OkResponse)(nil)

func NewOkResponse() *OkResponse {
	return &OkResponse{}
}

func (o *OkResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	render.Status(r, http.StatusOK)

	return nil
}
