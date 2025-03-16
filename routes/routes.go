package routes

import (
	"context"
	"net/http"

	"code.gitea.io/sdk/gitea"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	"OJ-API/config"
	_ "OJ-API/docs"
	"OJ-API/handlers"
)

type contextKey string

const clientContextKey contextKey = "client"
const userContextKey contextKey = "user"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if c, err := gitea.NewClient("http://"+config.Config("GITEA_HOST"), gitea.SetToken(token)); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		} else {
			ctx := context.WithValue(r.Context(), clientContextKey, c)
			if u, _, err := c.GetMyUserInfo(); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			} else {
				ctx = context.WithValue(ctx, userContextKey, u)
				next.ServeHTTP(w, r.WithContext(ctx))
			}
		}
	})
}

// New create an instance of Book app routes
func New() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), //The url pointing to API definition
	))

	r.Post("/api/sandbox", handlers.PostSandboxCmd)
	r.Get("/api/sandbox/status", handlers.GetSandboxStatus)
	// r.Get("/api/scores", handlers.GetScores)
	r.Get("/api/score", handlers.GetScoreByRepo)
	r.Post("/api/gitea", handlers.PostGiteaHook)
	return r
}
