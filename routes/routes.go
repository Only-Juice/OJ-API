package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	"OJ-API/config"
	_ "OJ-API/docs"
	"OJ-API/handlers"
	"OJ-API/models"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(handlers.ResponseHTTP{
				Success: false,
				Message: "Missing Token",
			})
			return
		}
		token = strings.TrimPrefix(token, "token ")
		if c, err := gitea.NewClient("http://"+config.Config("GIT_HOST"), gitea.SetToken(token)); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else {
			ctx := context.WithValue(r.Context(), models.ClientContextKey, c)
			if u, _, err := c.GetMyUserInfo(); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			} else {
				ctx = context.WithValue(ctx, models.UserContextKey, u)
				next.ServeHTTP(w, r.WithContext(ctx))
			}
		}
	})
}

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
	r.Post("/api/gitea/auth", handlers.PostBasicAuthenticationGitea)
	r.With(AuthMiddleware).Post("/api/gitea/user/bulk", handlers.PostBulkCreateUserGitea)
	r.With(AuthMiddleware).Post("/api/gitea/question/{question_id}", handlers.PostCreateQuestionRepositoryGitea)
	r.Get("/api/question", handlers.GetQuestionList)
	r.With(AuthMiddleware).Get("/api/question/user", handlers.GetUsersQuestions)
	r.With(AuthMiddleware).Get("/api/question/{UQR_ID}", handlers.GetQuestion)
	return r
}
