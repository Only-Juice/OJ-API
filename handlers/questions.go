package handlers

import (
	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"
)

type _GetQuestionListQuestionData struct {
	models.Question
	HasQuestion bool `json:"has_question"`
}

type GetQuestionListResponseData struct {
	QuestionCount int                            `json:"question_count" validate:"required"`
	Questions     []_GetQuestionListQuestionData `json:"questions" validate:"required"`
}

// GetQuestionList is a function to get a list of questions
// @Summary		Get a list of questions [Optional Authentication]
// @Description	Get a list of questions. Authentication is optional - if authenticated, shows user's question status.
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			page	query	int		false	"page number of results to return (1-based)"
// @Param			limit	query	int		false	"page size of results. Default is 10."
// @Param			status	query	string	false	"Filter by question status: 'all', 'active', or 'expired'. Default is 'all'."
// @Success		200		{object}	ResponseHTTP{data=[]GetQuestionListResponseData}
// @Failure		404
// @Failure		503
// @Router			/api/question [get]
// @Security		BearerAuth
func GetQuestionList(c *gin.Context) {
	db := database.DBConn
	jwtClaim, ok := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	var userID uint
	if ok && jwtClaim != nil {
		userID = jwtClaim.UserID
	}

	// Parse query parameters for pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.DefaultQuery("status", "all") // all, active, expired

	// Calculate offset
	offset := (page - 1) * limit

	// Build base query
	baseQuery := db.Model(&models.Question{}).
		Where("id NOT IN (SELECT question_id FROM exam_questions)")

	// Add status filter
	now := time.Now()
	switch status {
	case "active":
		baseQuery = baseQuery.Where("start_time <= ? AND end_time >= ?", now, now)
	case "expired":
		baseQuery = baseQuery.Where("end_time < ?", now)
		// "all" doesn't add any additional filter
	}

	var totalQuestions int64
	baseQuery.Count(&totalQuestions)

	var questions []_GetQuestionListQuestionData

	// Build select query with sorting
	selectQuery := "questions.*, CASE WHEN user_question_relations.id IS NOT NULL THEN true ELSE false END AS has_question"
	if userID == 0 {
		selectQuery = "questions.*, false AS has_question"
	}

	// Sort by status: active questions first, then expired
	orderClause := "CASE WHEN start_time <= '" + now.Format("2006-01-02 15:04:05") + "' AND end_time >= '" + now.Format("2006-01-02 15:04:05") + "' THEN 0 ELSE 1 END, start_time DESC, end_time ASC"

	if userID != 0 {
		query := db.Table("questions").Select(selectQuery).
			Joins("LEFT JOIN user_question_relations ON questions.id = user_question_relations.question_id AND user_question_relations.user_id = ?", userID).
			Where("questions.id NOT IN (SELECT question_id FROM exam_questions)")

		// Add status filter to main query
		switch status {
		case "active":
			query = query.Where("questions.start_time <= ? AND questions.end_time >= ?", now, now)
		case "expired":
			query = query.Where("questions.end_time < ?", now)
		}

		query.Order(orderClause).
			Offset(offset).Limit(limit).Scan(&questions)
	} else {
		query := db.Table("questions").Select(selectQuery).
			Where("questions.id NOT IN (SELECT question_id FROM exam_questions)")

		// Add status filter to main query
		switch status {
		case "active":
			query = query.Where("questions.start_time <= ? AND questions.end_time >= ?", now, now)
		case "expired":
			query = query.Where("questions.end_time < ?", now)
		}

		query.Order(orderClause).
			Offset(offset).Limit(limit).Scan(&questions)
	}

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
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
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

	client, err := gitea.NewClient(config.GetGiteaBaseURL())
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
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
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

type AddQuestionRequest struct {
	Title       string    `json:"title" validate:"required" example:"Question Title"`
	Description string    `json:"description" validate:"required" example:"Question Description"`
	GitRepoURL  string    `json:"git_repo_url" validate:"required" example:"user_name/repo_name"`
	StartTime   time.Time `json:"start_time" example:"2006-01-02T15:04:05Z" time_format:"RFC3339"`
	EndTime     time.Time `json:"end_time" example:"2006-01-02T15:04:05Z" time_format:"RFC3339"`
}

// AddQuestion is a function to add a question
// @Summary		Add a question
// @Description	Add a question
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			question	body		AddQuestionRequest	true	"Question object"
// @Success		200		{object}	ResponseHTTP{data=models.Question}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router			/api/question [post]
// @Security		BearerAuth
func AddQuestion(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	var question AddQuestionRequest
	if err := c.ShouldBindJSON(&question); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to parse question",
		})
		return
	}

	// Check if question with same GitRepoURL already exists
	var existingQuestion models.Question
	if err := db.Where("git_repo_url = ?", question.GitRepoURL).First(&existingQuestion).Error; err == nil {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "Question with this GitRepoURL already exists",
		})
		return
	}

	if err := db.Create(&models.Question{
		Title:       question.Title,
		Description: question.Description,
		GitRepoURL:  question.GitRepoURL,
		StartTime:   question.StartTime,
		EndTime:     question.EndTime,
	}).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to create question",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Question created successfully",
		Data:    question,
	})
}

type PatchQuestionRequest struct {
	Title       string    `json:"title" example:"Question Title"`
	Description string    `json:"description" example:"Question Description"`
	GitRepoURL  string    `json:"git_repo_url" example:"user_name/repo_name"`
	StartTime   time.Time `json:"start_time" example:"2006-01-02T15:04:05Z" time_format:"RFC3339"`
	EndTime     time.Time `json:"end_time" example:"2006-01-02T15:04:05Z" time_format:"RFC3339"`
}

// PatchQuestion is a function to update a question
// @Summary		Update a question
// @Description	Update a question
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			question	body		AddQuestionRequest	true	"Question object"
// @Param			ID		path		int				true	"ID of the Question to update"
// @Success		200		{object}	ResponseHTTP{data=models.Question}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router			/api/question/id/{ID} [patch]
// @Security		BearerAuth
func PatchQuestion(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Unauthorized",
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
	if err := db.Where("id = ?", ID).First(&question).Error; err != nil {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	var updateQuestion PatchQuestionRequest
	if err := c.ShouldBindJSON(&updateQuestion); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to parse question",
		})
		return
	}

	if updateQuestion.Title != "" {
		question.Title = updateQuestion.Title
	}
	if updateQuestion.Description != "" {
		question.Description = updateQuestion.Description
	}
	if updateQuestion.GitRepoURL != "" {
		question.GitRepoURL = updateQuestion.GitRepoURL
	}
	if !updateQuestion.StartTime.IsZero() {
		question.StartTime = updateQuestion.StartTime
	}
	if !updateQuestion.EndTime.IsZero() {
		question.EndTime = updateQuestion.EndTime
	}

	if err := db.Save(&question).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to update question",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Question updated successfully",
		Data:    question,
	})
}

// DeleteQuestion is a function to delete a question
// @Summary		Delete a question
// @Description	Delete a question
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			ID	path	int	true	"ID of the Question to delete"
// @Success		200		{object}	ResponseHTTP{data=models.Question}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router			/api/question/id/{ID} [delete]
// @Security		BearerAuth
func DeleteQuestion(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Unauthorized",
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
	if err := db.Where("id = ?", ID).First(&question).Error; err != nil {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	if err := db.Delete(&question).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to delete question",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Question deleted successfully",
		Data:    question,
	})
}

type QuestionTestScript struct {
	TestScript string `json:"test_script" validate:"required"`
}

// GetQuestionTestScript is a function to get the test script for a question
// @Summary		Get the test script for a question
// @Description	Get the test script for a question
// @Tags			Question
// @Accept			json
// @Produce		json
// @Param			ID	path	int	true	"ID of the Question to get the test script for"
// @Success		200		{object}	ResponseHTTP{data=QuestionTestScript}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router			/api/question/test_script/{ID} [get]
// @Security		BearerAuth
func GetQuestionTestScript(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Unauthorized",
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

	var questionTestScript models.QuestionTestScript
	if err := db.Where("question_id = ?", ID).First(&questionTestScript).Error; err != nil {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Question test script not found",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Question test script fetched successfully",
		Data: QuestionTestScript{
			TestScript: questionTestScript.TestScript,
		},
	})
}
