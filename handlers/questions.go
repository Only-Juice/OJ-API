package handlers

import (
	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
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
// @Failure		404
// @Failure		503
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
	if len(questions) == 0 {
		c.JSON(404, ResponseHTTP{
			Success: true,
			Message: "No questions found",
			Data:    nil,
		})
		return
	}

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
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router			/api/question/user [get]
// @Security		BearerAuth
func GetUsersQuestions(c *gin.Context) {
	db := database.DBConn
	jwtClaim := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	userID := jwtClaim.UserID

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

func GetReadme(client *gitea.Client, userName string, gitRepoURL string) string {
	branches := []string{"main", "master"}
	readmeFiles := []string{"README.md", "README"}

	for _, branch := range branches {
		for _, readmeFile := range readmeFiles {
			fileContent, _, err := client.GetFile(userName, gitRepoURL, branch, readmeFile)
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
// @Success		200		{object}	ResponseHTTP{data=GetQuestionResponseData}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router			/api/question/{UQR_ID} [get]
// @Security		BearerAuth
func GetQuestion(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient("http://"+config.Config("GIT_HOST"),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

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
	db.Where("id = ? AND user_id = ?", UQR_ID, jwtClaims.UserID).Limit(1).Find(&uqr)
	if uqr.ID == 0 {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	var question models.Question
	db.Where("id = ?", uqr.QuestionID).Limit(1).Find(&question)
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
			README:           GetReadme(client, jwtClaims.Username, strings.Split(uqr.GitUserRepoURL, "/")[1]),
			GitRepoURL:       uqr.GitUserRepoURL,
			ParentGitRepoURL: question.GitRepoURL,
		},
	})
}

// GetQuestionByID is a function to get a question by ID
// @Summary		Get a question by ID
// @Description Retrieve only public questions
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			ID	path	int	true	"ID of the Question to get"
// @Success		200		{object}	ResponseHTTP{data=GetQuestionResponseData}
// @Failure		404
// @Failure		503
// @Router			/api/question/id/{ID} [get]
func GetQuestionByID(c *gin.Context) {
	db := database.DBConn

	client, err := gitea.NewClient("http://" + config.Config("GIT_HOST"))
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	IDstr := c.Param("ID")
	ID, err := strconv.Atoi(IDstr)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Invalid ID",
		})
		return
	}

	var question models.Question
	db.Where("id = ?", ID).Limit(1).Find(&question)
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
			GitRepoURL:       question.GitRepoURL,
			ParentGitRepoURL: "",
			README:           GetReadme(client, strings.Split(question.GitRepoURL, "/")[0], strings.Split(question.GitRepoURL, "/")[1]),
		},
	})
}

type GetUserQuestionResponseData struct {
	GetQuestionResponseData
	UQR_ID uint `json:"uqr_id" validate:"required"`
}

// GetUserQuestionByID is a function to get a user's question by Question ID
// @Summary		Get a user's question by Question ID
// @Description	Retrieve a specific question associated with a user by its Question ID
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			ID	path	int	true	"ID of the Question to get"
// @Success		200		{object}	ResponseHTTP{data=GetUserQuestionResponseData}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router			/api/question/user/id/{ID} [get]
// @Security		BearerAuth
func GetUserQuestionByID(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient("http://"+config.Config("GIT_HOST"),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	IDstr := c.Param("ID")
	ID, err := strconv.Atoi(IDstr)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Invalid ID",
		})
		return
	}

	var uqr models.UserQuestionRelation
	db.Where("question_id = ? AND user_id = ?", ID, jwtClaims.UserID).Limit(1).Find(&uqr)
	if uqr.ID == 0 {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	var question models.Question
	db.Where("id = ?", uqr.QuestionID).Limit(1).Find(&question)
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
		Data: GetUserQuestionResponseData{
			GetQuestionResponseData: GetQuestionResponseData{
				Title:            question.Title,
				Description:      question.Description,
				README:           GetReadme(client, jwtClaims.Username, strings.Split(uqr.GitUserRepoURL, "/")[1]),
				GitRepoURL:       uqr.GitUserRepoURL,
				ParentGitRepoURL: question.GitRepoURL,
			},
			UQR_ID: uqr.ID,
		},
	})
}
