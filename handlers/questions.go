package handlers

import (
	"OJ-API/database"
	"OJ-API/models"
	"encoding/json"
	"fmt"
	"net/http"
)

type GetQuestionListResponseData struct {
	QuestionCount int               `json:"question_count" validate:"required"`
	Questions     []models.Question `json:"questions" validate:"required"`
}

// GetQuestionList is a function to get a list of questions
// @Summary		Get a list of questions
// @Description	Get a list of questions
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			page	query	int		false	"page number of results to return (1-based)"
// @Param			limit	query	int		false	"page size of results. Default is 10."
// @Success		200		{object}	ResponseHTTP{data=[]GetQuestionListResponseData}
// @Failure		404		{object}	ResponseHTTP{}
// @Failure		503		{object}	ResponseHTTP{}
// @Router			/api/question [get]
func GetQuestionList(w http.ResponseWriter, r *http.Request) {
	db := database.DBConn

	// Parse query parameters for pagination
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	// Default values for page and limit
	pageNum := 1
	limitNum := 10

	if page != "" {
		fmt.Sscanf(page, "%d", &pageNum)
	}
	if limit != "" {
		fmt.Sscanf(limit, "%d", &limitNum)
	}

	// Calculate offset
	offset := (pageNum - 1) * limitNum

	var totalQuestions int64
	db.Model(&models.Question{}).Count(&totalQuestions)
	var questions []models.Question
	db.Offset(offset).Limit(limitNum).Find(&questions)

	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Message: "Questions fetched successfully",
		Data:    GetQuestionListResponseData{QuestionCount: int(totalQuestions), Questions: questions},
	})
}
