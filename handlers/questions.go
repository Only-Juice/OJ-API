package handlers

import (
	"OJ-API/database"
	"OJ-API/models"
	"encoding/json"
	"fmt"
	"net/http"

	"code.gitea.io/sdk/gitea"
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

type _GetQuestionResponseData struct {
	GitRepoUrl       string `json:"git_repo_url" validate:"required"`
	ParentGitRepoUrl string `json:"parent_git_repo_url" validate:"required"`
	Title            string `json:"title" validate:"required"`
	Description      string `json:"description" validate:"required"`
	UQRID            uint   `json:"uqr_id" validate:"required"`
	QID              uint   `json:"q_id" validate:"required"`
}

type GetQuestionResponseData struct {
	QuestionCount int                        `json:"question_count" validate:"required"`
	Questions     []_GetQuestionResponseData `json:"question" validate:"required"`
}

// GetUsersQuestions is a function to get a list of questions by user
// @Summary		Get a list of questions by user
// @Description	Get a list of questions by user
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			page	query	int		false	"page number of results to return (1-based)"
// @Param			limit	query	int		false	"page size of results. Default is 10."
// @Success		200		{object}	ResponseHTTP{data=[]GetQuestionResponseData}
// @Failure		404		{object}	ResponseHTTP{}
// @Failure		503		{object}	ResponseHTTP{}
// @Router			/api/question/user [get]
// @Security		AuthorizationHeaderToken
func GetUsersQuestions(w http.ResponseWriter, r *http.Request) {
	db := database.DBConn
	user := r.Context().Value(models.UserContextKey).(*gitea.User)
	userID := user.ID

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
	db.Model(&models.UserQuestionRelation{}).Where("user_id = ?", userID).Count(&totalQuestions)
	var questions []struct {
		models.Question
		UQRID          uint
		GitUserRepoURL string
	}
	db.Table("questions").Select("questions.*, user_question_relations.id as uqr_id, user_question_relations.git_user_repo_url").Joins("JOIN user_question_relations ON questions.id = user_question_relations.question_id").Where("user_question_relations.user_id = ?", userID).Offset(offset).Limit(limitNum).Scan(&questions)

	var responseQuestions []_GetQuestionResponseData
	for _, question := range questions {
		responseQuestions = append(responseQuestions, _GetQuestionResponseData{
			GitRepoUrl:       question.GitUserRepoURL,
			ParentGitRepoUrl: question.GitRepoURL,
			Title:            question.Title,
			Description:      question.Description,
			UQRID:            question.UQRID,
			QID:              question.ID,
		})
	}
	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Message: "Questions fetched successfully",
		Data: GetQuestionResponseData{
			QuestionCount: int(totalQuestions),
			Questions:     responseQuestions,
		},
	})
}
