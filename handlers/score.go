package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"gorm.io/gorm"

	"OJ-API/database"
	"OJ-API/models"
)

// GetScores is a function to get all scores
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
//	@Summary		Get a score by repo
//	@Description	Get a score by repo
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Param			repo	query		string	true	"Repo name"
//	@Success		200		{object}	ResponseHTTP{data=models.Score}
//	@Failure		404		{object}	ResponseHTTP{}
//	@Failure		503		{object}	ResponseHTTP{}
//	@Router			/api/score [get]
func GetScoreByRepo(w http.ResponseWriter, r *http.Request) {
	db := database.DBConn
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
	var score models.Score
	if err := db.Where("git_repo = ?", repo).Order("updated_at DESC").First(&score).Error; err != nil {
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Message: "Successfully get score by repo",
		Data:    score,
	})
}