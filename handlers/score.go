package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"gorm.io/gorm"

	"OJ-API/database"
	"OJ-API/models"
)

// GetScores is a function to get all scores
//
//	@Summary		Get all scores
//	@Description	Get all scores
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	ResponseHTTP{data=[]models.Score}
//	@Failure		503		{object}	ResponseHTTP{}
//	@Router			/api/scores [get]
func GetScores(w http.ResponseWriter, r *http.Request) {
	db := database.DBConn
	var scores []models.Score
	if err := db.Raw(`
		SELECT * FROM scores
		WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY git_repo ORDER BY updated_at DESC) AS rn
				FROM scores
			) AS subquery
			WHERE rn = 1
		)
		ORDER BY updated_at DESC
	`).Scan(&scores).Error; err != nil {
		log.Printf("Failed to get scores: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Failed to get scores",
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Message: "Successfully get scores",
		Data:    scores,
	})
}

// GetScoreByRepo is a function to get a score by repo
//
//	@Summary		Get a score by repo
//	@Description	Get a score by repo
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Param			owner	query	string	true	"owner of the repo"
//	@Param			repo		query	string	true	"name of the repo"
//	@Param			page		query	int		false	"page number of results to return (1-based)"
//	@Param			limit		query	int		false	"page size of results. Default is 10."
//	@Success		200		{object}	ResponseHTTP{data=models.Score}
//	@Failure		404		{object}	ResponseHTTP{}
//	@Failure		503		{object}	ResponseHTTP{}
//	@Router			/api/score [get]
func GetScoreByRepo(w http.ResponseWriter, r *http.Request) {
	db := database.DBConn
	owner, err := url.QueryUnescape(r.URL.Query().Get("owner"))
	if err != nil {
		log.Printf("Failed to unescape repo name: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Failed to unescape user name",
		})
		return
	}
	if owner == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "User name is required",
		})
		return
	}
	repo, err := url.QueryUnescape(r.URL.Query().Get("repo"))
	if err != nil {
		log.Printf("Failed to unescape repo name: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Failed to unescape repo name",
		})
		return
	}
	if repo == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Repo name is required",
		})
		return
	}
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	if page == "" {
		page = "1"
	}
	if limit == "" {
		limit = "10"
	}
	var scores []models.Score
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		log.Printf("Invalid page number: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Invalid page number",
		})
		return
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		log.Printf("Invalid limit number: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Invalid limit number",
		})
		return
	}
	offset := (pageInt - 1) * limitInt

	if err := db.Where("user_name = ? AND repo_name = ?", owner, repo).
		Order("updated_at DESC").
		Offset(offset).
		Limit(limitInt).
		Find(&scores).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ResponseHTTP{
				Success: false,
				Message: "Score not found",
			})
			return
		}
		log.Printf("Failed to get score by repo: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Failed to get score by repo",
		})
		return
	}
	if len(scores) == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Score not found",
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Message: "Successfully get score by repo",
		Data:    scores,
	})
}
