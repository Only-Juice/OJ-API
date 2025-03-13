package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"gorm.io/gorm"

	"OJ-API/database"
	"OJ-API/models"
)

type Score struct {
	Score     float64   `json:"score" example:"100" validate:"required"`
	Message   string    `json:"message" example:"Scored successfully" validate:"required"`
	JudgeTime time.Time `json:"judge_time" example:"2021-07-01T00:00:00Z" validate:"required"`
}

type GetScoreResponseData struct {
	ScoresCount int     `json:"scores_count" validate:"required"`
	Scores      []Score `json:"scores" validate:"required"`
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
//	@Success		200		{object}	ResponseHTTP{data=GetScoreResponseData}
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
	var _scores []models.UserQuestionTable
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

	repoURL := fmt.Sprintf("%s/%s", owner, repo)
	var totalCount int64
	if err := db.Model(&models.UserQuestionTable{}).
		Joins("UQR").
		Where("git_user_repo_url = ?", repoURL).
		Count(&totalCount).Error; err != nil {
		log.Printf("Failed to count scores: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Failed to count scores",
		})
		return
	}

	if err := db.Model(&models.UserQuestionTable{}).
		Joins("UQR").
		Where("git_user_repo_url = ?", repoURL).
		Offset(offset).
		Limit(limitInt).
		Find(&_scores).Error; err != nil {
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
	if len(_scores) == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Score not found",
		})
		return
	}
	var scores []Score
	for _, score := range _scores {
		scores = append(scores, Score{
			Score:     score.Score,
			Message:   score.Message,
			JudgeTime: score.JudgeTime,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Message: "Successfully get score by repo",
		Data: GetScoreResponseData{
			Scores:      scores,
			ScoresCount: int(totalCount),
		},
	})
}
