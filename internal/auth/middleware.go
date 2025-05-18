package auth

import (
	"errors"
	"net/http"
	"slices"

	"github.com/AlejandroHerr/go-common/pkg/api"
	"github.com/go-chi/render"
)

func ValidateToken(authTokens []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")

			if token == "" || !slices.Contains(authTokens, token) {
				resp := api.NewErrorResponse(
					errors.New("Unauthorized"),
					http.StatusUnauthorized,
					http.StatusText(http.StatusUnauthorized),
					"Unauthorized",
					nil,
				)

				if err := render.Render(w, r, resp); err != nil {
					render.Render(w, r, api.RenderErrorResponse(err)) //nolint: errcheck,gosec // ignore error
				}

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
