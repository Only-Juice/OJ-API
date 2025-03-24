package handlers

import (
	"OJ-API/database"
	"OJ-API/models"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"
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
func GetQuestionList(c *gin.Context) {
	db := database.DBConn

	// Parse query parameters for pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Calculate offset
	offset := (page - 1) * limit

	var totalQuestions int64
	db.Model(&models.Question{}).Count(&totalQuestions)
	var questions []models.Question
	db.Offset(offset).Limit(limit).Find(&questions)

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Questions fetched successfully",
		Data: GetQuestionListResponseData{
			QuestionCount: int(totalQuestions),
			Questions:     questions,
		},
	})
}

type _GetUsersQuestionsResponseData struct {
	GitRepoUrl       string `json:"git_repo_url" validate:"required"`
	ParentGitRepoUrl string `json:"parent_git_repo_url" validate:"required"`
	Title            string `json:"title" validate:"required"`
	Description      string `json:"description" validate:"required"`
	UQRID            uint   `json:"uqr_id" validate:"required"`
	QID              uint   `json:"q_id" validate:"required"`
}

type GetUsersQuestionsResponseData struct {
	QuestionCount int                              `json:"question_count" validate:"required"`
	Questions     []_GetUsersQuestionsResponseData `json:"question" validate:"required"`
}

// GetUsersQuestions is a function to get a list of questions by user
// @Summary		Get a list of questions by user
// @Description	Get a list of questions by user
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			page	query	int		false	"page number of results to return (1-based)"
// @Param			limit	query	int		false	"page size of results. Default is 10."
// @Success		200		{object}	ResponseHTTP{data=[]GetUsersQuestionsResponseData}
// @Failure		404		{object}	ResponseHTTP{}
// @Failure		503		{object}	ResponseHTTP{}
// @Router			/api/question/user [get]
// @Security		AuthorizationHeaderToken
func GetUsersQuestions(c *gin.Context) {
	db := database.DBConn
	giteaUser := c.Request.Context().Value(models.UserContextKey).(*gitea.User)
	user := models.User{UserName: giteaUser.UserName}
	db.Where(&user).First(&user)
	userID := user.ID

	// Parse query parameters for pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Calculate offset
	offset := (page - 1) * limit

	var totalQuestions int64
	db.Model(&models.UserQuestionRelation{}).Where("user_id = ?", userID).Count(&totalQuestions)
	var questions []struct {
		models.Question
		UQRID          uint
		GitUserRepoURL string
	}
	db.Table("questions").Select("questions.*, user_question_relations.id as uqr_id, user_question_relations.git_user_repo_url").
		Joins("JOIN user_question_relations ON questions.id = user_question_relations.question_id").
		Where("user_question_relations.user_id = ?", userID).
		Offset(offset).Limit(limit).Scan(&questions)

	if len(questions) == 0 {
		c.JSON(404, ResponseHTTP{
			Success: true,
			Message: "No questions found",
			Data: GetUsersQuestionsResponseData{
				QuestionCount: 0,
				Questions:     []_GetUsersQuestionsResponseData{},
			},
		})
		return
	}

	var responseQuestions []_GetUsersQuestionsResponseData
	for _, question := range questions {
		responseQuestions = append(responseQuestions, _GetUsersQuestionsResponseData{
			GitRepoUrl:       question.GitUserRepoURL,
			ParentGitRepoUrl: question.GitRepoURL,
			Title:            question.Title,
			Description:      question.Description,
			UQRID:            question.UQRID,
			QID:              question.ID,
		})
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Questions fetched successfully",
		Data: GetUsersQuestionsResponseData{
			QuestionCount: int(totalQuestions),
			Questions:     responseQuestions,
		},
	})
}

type GetQuestionResponseData struct {
	Title            string `json:"title" validate:"required"`
	Description      string `json:"description" validate:"required"`
	README           string `json:"readme" validate:"required"`
	GitRepoURL       string `json:"git_repo_url" validate:"required"`
	ParentGitRepoURL string `json:"parent_git_repo_url" validate:"required"`
}

func GetReadme(client *gitea.Client, user *gitea.User, gitRepoURL string) string {
	branches := []string{"main", "master"}
	readmeFiles := []string{"README.md", "README"}

	for _, branch := range branches {
		for _, readmeFile := range readmeFiles {
			fileContent, _, err := client.GetFile(user.UserName, gitRepoURL, branch, readmeFile)
			if err == nil {
				return string(fileContent)
			}
		}
	}
	return ""
}

// GetQuestion is a function to get a question by UQR_ID
// @Summary		Get a question by UQR_ID
// @Description	Get a question by UQR_ID
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			UQR_ID	path	int	true	"ID of the UserQuestionRelation to get"
// @Success		200		{object}	ResponseHTTP{}
// @Failure		404		{object}	ResponseHTTP{}
// @Failure		503		{object}	ResponseHTTP{}
// @Router			/api/question/{UQR_ID} [get]
// @Security		AuthorizationHeaderToken
func GetQuestion(c *gin.Context) {
	db := database.DBConn
	client := c.Request.Context().Value(models.ClientContextKey).(*gitea.Client)
	giteaUser := c.Request.Context().Value(models.UserContextKey).(*gitea.User)
	user := models.User{UserName: giteaUser.UserName}
	db.Where(&user).First(&user)
	userID := user.ID

	UQR_IDstr := c.Param("UQR_ID")
	UQR_ID, err := strconv.Atoi(UQR_IDstr)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Invalid UQR ID",
		})
		return
	}

	var uqr models.UserQuestionRelation
	db.Where("id = ? AND user_id = ?", UQR_ID, userID).First(&uqr)
	if uqr.ID == 0 {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	var question models.Question
	db.Where("id = ?", uqr.QuestionID).First(&question)
	if question.ID == 0 {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Question fetched successfully",
		Data: GetQuestionResponseData{
			Title:            question.Title,
			Description:      question.Description,
			README:           GetReadme(client, giteaUser, strings.Split(uqr.GitUserRepoURL, "/")[1]),
			GitRepoURL:       uqr.GitUserRepoURL,
			ParentGitRepoURL: question.GitRepoURL,
		},
	})
}
