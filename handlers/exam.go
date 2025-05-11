package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
)

type ExamRequest struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time" example:"2006-01-02T15:04:05Z" time_format:"RFC3339"`
	EndTime     time.Time `json:"end_time" example:"2006-01-02T15:04:05Z" time_format:"RFC3339"`
}

// CreateExam handles the creation of a new exam
// @Summary      Create a new exam
// @Description  Create a new exam with the provided details
// @Tags         Exam
// @Accept       json
// @Produce      json
// @Param        exam body ExamRequest true "Exam details"
// @Success      200 {object} ResponseHTTP{data=models.Exam}
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router       /api/exams [post]
// @Security BearerAuth
func CreateExam(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}

	var exam ExamRequest
	if err := c.ShouldBindJSON(&exam); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid input: " + err.Error(),
		})
		return
	}

	// Ensure StartTime is before EndTime
	if exam.StartTime.After(exam.EndTime) {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Start time must be before end time",
		})
		return
	}

	db := database.DBConn
	newExam := models.Exam{
		Title:       exam.Title,
		Description: exam.Description,
		StartTime:   exam.StartTime,
		EndTime:     exam.EndTime,
		OwnerID:     jwtClaims.UserID,
	}
	if err := db.Create(&newExam).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to create exam",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    newExam,
	})
}

// GetExam retrieves an exam by ID
// @Summary      Get an exam by ID
// @Description  Retrieve an exam's details using its ID
// @Tags         Exam
// @Produce      json
// @Param        id path string true "Exam ID"
// @Success      200 {object} ResponseHTTP{data=models.Exam}
// @Failure      404 {object} ResponseHTTP{}
// @Router       /api/exams/{id} [get]
// @Security BearerAuth
func GetExam(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}

	id := c.Param("id")
	var exam models.Exam

	db := database.DBConn
	if err := db.First(&exam, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Exam not found",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    exam,
	})
}

// UpdateExam updates an existing exam
// @Summary      Update an existing exam
// @Description  Update the details of an existing exam by ID
// @Tags         Exam
// @Accept       json
// @Produce      json
// @Param        id path string true "Exam ID"
// @Param        exam body models.Exam true "Updated exam details"
// @Success      200 {object} ResponseHTTP{data=models.Exam}
// @Failure      400 {object} ResponseHTTP{}
// @Failure      404 {object} ResponseHTTP{}
// @Failure      500 {object} ResponseHTTP{}
// @Router       /api/exams/{id} [put]
// @Security BearerAuth
func UpdateExam(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}

	id := c.Param("id")
	var exam models.Exam

	db := database.DBConn
	if err := db.First(&exam, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Exam not found",
		})
		return
	}

	if err := c.ShouldBindJSON(&exam); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	if err := db.Save(&exam).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to update exam",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    exam,
	})
}

// DeleteExam deletes an exam by ID
// @Summary      Delete an exam by ID
// @Description  Delete an exam using its ID
// @Tags         Exam
// @Produce      json
// @Param        id path string true "Exam ID"
// @Success      200 {object} ResponseHTTP{}
// @Failure      404 {object} ResponseHTTP{}
// @Failure      500 {object} ResponseHTTP{}
// @Router       /api/exams/{id} [delete]
// @Security BearerAuth
func DeleteExam(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}

	id := c.Param("id")
	var exam models.Exam

	db := database.DBConn
	if err := db.First(&exam, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Exam not found",
		})
		return
	}

	if err := db.Delete(&exam).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to delete exam",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "Exam deleted successfully",
	})
}

type ExamListData struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time" example:"2006-01-02T15:04:05Z" time_format:"RFC3339"`
	EndTime     time.Time `json:"end_time" example:"2006-01-02T15:04:05Z" time_format:"RFC3339"`
}

// ListExams retrieves all exams
// @Summary      List all exams
// @Description  Retrieve a list of all exams
// @Tags         Exam
// @Produce      json
// @Success      200 {object} ResponseHTTP{data=[]ExamListData}
// @Failure      500 {object} ResponseHTTP{}
// @Router       /api/exams [get]
func ListExams(c *gin.Context) {
	var exams []models.Exam

	db := database.DBConn
	if err := db.Find(&exams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve exams",
		})
		return
	}

	examListData := make([]ExamListData, len(exams))
	for i, exam := range exams {
		examListData[i] = ExamListData{
			ID:          exam.ID,
			Title:       exam.Title,
			Description: exam.Description,
			StartTime:   exam.StartTime,
			EndTime:     exam.EndTime,
		}
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    examListData,
	})
}

// GetExamQuestions retrieves all questions for a specific exam
// @Summary      Get questions for an exam
// @Description  Retrieve all questions associated with a specific exam
// @Tags         Exam
// @Produce      json
// @Param        id path string true "Exam ID"
// @Success      200 {object} ResponseHTTP{data=[]models.Question}
// @Failure      404 {object} ResponseHTTP{}
// @Failure      500 {object} ResponseHTTP{}
// @Router       /api/exams/{id}/questions [get]
func GetExamQuestions(c *gin.Context) {
	id := c.Param("id")
	var exam models.Exam

	db := database.DBConn
	if err := db.First(&exam, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Exam not found",
		})
		return
	}

	var examQuestions []models.ExamQuestion
	if err := db.Where(&models.ExamQuestion{
		ExamID: exam.ID,
	}).Joins("Question").Find(&examQuestions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve questions: " + err.Error(),
		})
		return
	}

	questions := make([]models.Question, len(examQuestions))
	for i, eq := range examQuestions {
		questions[i] = eq.Question
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    questions,
	})
}

type point struct {
	Score int `json:"score" binding:"required"`
}

// AddQuestionToExam adds a question to an exam
// @Summary      Add a question to an exam
// @Description  Associate a question with a specific exam
// @Tags         Exam
// @Accept       json
// @Produce      json
// @Param        id path string true "Exam ID"
// @Param        question_id path string true "Question ID"
// @Param		 point body point true "Score for the question"
// @Success      200 {object} ResponseHTTP{}
// @Failure      404 {object} ResponseHTTP{}
// @Failure      500 {object} ResponseHTTP{}
// @Router       /api/exams/{id}/questions/{question_id} [post]
// @Security BearerAuth
func AddQuestionToExam(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}

	id := c.Param("id")
	var exam models.Exam

	db := database.DBConn
	if err := db.First(&exam, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Exam not found",
		})
		return
	}

	questionID := c.Param("question_id")
	var question models.Question
	if err := db.First(&question, questionID).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	point := point{}
	if err := c.ShouldBindJSON(&point); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid input: " + err.Error(),
		})
		return
	}
	if point.Score <= 0 {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Score must be greater than 0",
		})
		return
	}
	// Check if the question is already associated with the exam
	var existingQuestion models.ExamQuestion
	if err := db.Where("exam_id = ? AND question_id = ?", exam.ID, question.ID).First(&existingQuestion).Error; err == nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Question already associated with the exam",
		})
		return
	}

	if err := db.Create(&models.ExamQuestion{
		ExamID:     exam.ID,
		QuestionID: question.ID,
		Point:      point.Score,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "Question added to exam successfully",
	})
}

// RemoveQuestionFromExam removes a question from an exam
// @Summary      Remove a question from an exam
// @Description  Disassociate a question from a specific exam
// @Tags         Exam
// @Produce      json
// @Param        id path string true "Exam ID"
// @Param        question_id path string true "Question ID"
// @Success      200 {object} ResponseHTTP{}
// @Failure      404 {object} ResponseHTTP{}
// @Failure      500 {object} ResponseHTTP{}
// @Router       /api/exams/{id}/questions/{question_id} [delete]
// @Security BearerAuth
func RemoveQuestionFromExam(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}

	id := c.Param("id")
	var exam models.Exam

	db := database.DBConn
	if err := db.First(&exam, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Exam not found",
		})
		return
	}

	questionID := c.Param("question_id")
	var question models.Question
	if err := db.First(&question, questionID).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	// Check if the question is associated with the exam
	var examQuestion models.ExamQuestion
	if err := db.Where("exam_id = ? AND question_id = ?", exam.ID, question.ID).First(&examQuestion).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Question not associated with the exam",
		})
		return
	}
	if err := db.Delete(&examQuestion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to remove question from exam",
		})
		return
	}
	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "Question removed from exam successfully",
	})
}

type TopExamScore struct {
	QuestionID     int       `json:"question_id" example:"1" validate:"required"`
	GitUserRepoURL string    `json:"git_user_repo_url" example:"owner/repo" validate:"required"`
	Score          float64   `json:"score" example:"100" validate:"required"`
	Point          int       `json:"point" example:"100" validate:"required"`
	Message        string    `json:"message" example:"Scored successfully" validate:"required"`
	JudgeTime      time.Time `json:"judge_time" example:"2006-01-02T15:04:05Z07:00" time_format:"RFC3339" validate:"required"`
}

type GetTopExamScoreResponseData struct {
	ScoresCount int            `json:"scores_count" validate:"required"`
	Scores      []TopExamScore `json:"scores" validate:"required"`
}

// Get the top scores of each question in the exam for a specific user
// @Summary      Get top scores for each question in an exam
// @Description  Retrieve the top scores for each question in a specific exam for a user
// @Tags         Exam
// @Produce      json
// @Param        id path string true "Exam ID"
// @Param			page	query	int	false	"page number of results to return (1-based)"
// @Param			limit	query	int	false	"page size of results. Default is 10."
// @Success		200	{object}	ResponseHTTP{data=GetTopExamScoreResponseData}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router			/api/exams/{id}/score/top [get]
// @Security		BearerAuth
func GetTopExamScore(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	id := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var totalCount int64

	subQuery := db.Model(&models.UserQuestionTable{}).
		Select("DISTINCT question_id").
		Joins("JOIN user_question_relations UQR ON user_question_tables.uqr_id = UQR.id").
		Where("question_id IN (SELECT question_id FROM exam_questions WHERE exam_id = ?)", id).
		Where("UQR.user_id = ?", jwtClaims.UserID)

	if err := db.Table("(?) AS sub", subQuery).
		Count(&totalCount).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to count scores",
		})
		return
	}

	var scores []TopExamScore
	if err := db.Model(&models.UserQuestionTable{}).
		Joins("JOIN user_question_relations UQR ON user_question_tables.uqr_id = UQR.id").
		Joins("JOIN exam_questions EQ ON UQR.question_id = EQ.question_id").
		Select("DISTINCT ON (UQR.question_id) UQR.question_id, git_user_repo_url, score, message, judge_time, EQ.point").
		Where("UQR.user_id = ?", jwtClaims.UserID).
		Order("UQR.question_id, score DESC").
		Order("judge_time DESC").
		Offset(offset).
		Limit(limit).
		Find(&scores).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, ResponseHTTP{
				Success: false,
				Message: "Score not found",
			})
			return
		}
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to get top score",
		})
		return
	}

	if len(scores) == 0 {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "No scores found",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Successfully retrieved top scores",
		Data: GetTopExamScoreResponseData{
			Scores:      scores,
			ScoresCount: int(totalCount),
		},
	})
}

type EnhancedQuestionScore struct {
	QuestionID     int     `json:"question_id"`
	QuestionTitle  string  `json:"question_title"`
	GitUserRepoURL string  `json:"git_user_repo_url"`
	Score          float64 `json:"score"`
	WeightedScore  float64 `json:"weighted_score"`
}

type EnhancedLeaderboardScore struct {
	UserName       string                  `json:"user_name"`
	TotalScore     float64                 `json:"total_score"`
	QuestionScores []EnhancedQuestionScore `json:"question_scores"`
}

type EnhancedGetLeaderboardResponseData struct {
	Count  int                        `json:"count"`
	Scores []EnhancedLeaderboardScore `json:"scores"`
}

// GetExamLeaderboard retrieves the leaderboard for an exam
// @Summary      	Get the leaderboard for an exam
// @Description  	Retrieve the leaderboard for a specific exam
// @Tags         	Exam
// @Accept			json
// @Produce		json
// @Param        	id path string true "Exam ID"
// @Param        	page query int false "Page number of results to return (1-based)"
// @Param        	limit query int false "Page size of results. Default is 10."
// @Success		200	{object}	ResponseHTTP{data=EnhancedGetLeaderboardResponseData}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router			/api/exams/{id}/leaderboard [get]
func GetExamLeaderboard(c *gin.Context) {
	db := database.DBConn

	id := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Check if exam exists
	var exam models.Exam
	if err := db.First(&exam, id).Error; err != nil {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Exam not found",
		})
		return
	}

	// Get total count of users who have scores for this exam
	var totalCount int64
	if err := db.Table("(SELECT DISTINCT user_id FROM user_question_relations "+
		"JOIN user_question_tables ON user_question_relations.id = user_question_tables.uqr_id "+
		"JOIN exam_questions ON user_question_relations.question_id = exam_questions.question_id "+
		"WHERE exam_questions.exam_id = ?) AS t", id).
		Count(&totalCount).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to count users with scores",
		})
		return
	}

	// Get users with their total scores for this exam
	type UserWithTotalScore struct {
		UserID     uint    `json:"user_id"`
		UserName   string  `json:"user_name"`
		IsPublic   bool    `json:"is_public"`
		TotalScore float64 `json:"total_score"`
	}

	var usersWithScores []UserWithTotalScore
	if err := db.Table(`(
        SELECT 
            UQR.user_id AS user_id, 
            MAX(user_question_tables.score) / 100 * EQ.point AS max_score, 
            UQR.question_id
        FROM user_question_tables
        JOIN user_question_relations UQR ON user_question_tables.uqr_id = UQR.id
        JOIN exam_questions EQ ON UQR.question_id = EQ.question_id
        WHERE EQ.exam_id = ?
        GROUP BY UQR.user_id, UQR.question_id, EQ.point
    ) AS subquery`, id).
		Joins("JOIN users ON users.id = subquery.user_id").
		Select("users.id AS user_id, users.user_name, users.is_public, SUM(max_score) AS total_score").
		Group("users.id, users.user_name, users.is_public").
		Order("total_score DESC").
		Offset(offset).
		Limit(limit).
		Find(&usersWithScores).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to get exam leaderboard users",
		})
		return
	}

	if len(usersWithScores) == 0 {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "No scores found for this exam",
		})
		return
	}

	// Get all user IDs to fetch their question scores
	var userIDs []uint
	for _, u := range usersWithScores {
		userIDs = append(userIDs, u.UserID)
	}

	// Query for individual question scores for these users
	type QuestionScoreDetail struct {
		UserID         uint    `json:"user_id"`
		QuestionID     int     `json:"question_id"`
		QuestionTitle  string  `json:"question_title"`
		GitUserRepoURL string  `json:"git_user_repo_url"`
		Score          float64 `json:"score"`
		Point          int     `json:"point"`
		WeightedScore  float64 `json:"weighted_score"`
	}

	var questionScores []QuestionScoreDetail
	subquery := db.Model(&models.UserQuestionTable{}).
		Select("UQR.user_id, UQR.question_id, MAX(user_question_tables.score) AS score, UQR.git_user_repo_url, EQ.point").
		Joins("JOIN user_question_relations UQR ON user_question_tables.uqr_id = UQR.id").
		Joins("JOIN exam_questions EQ ON UQR.question_id = EQ.question_id").
		Where("UQR.user_id IN ?", userIDs).
		Where("EQ.exam_id = ?", id).
		Group("UQR.user_id, UQR.question_id, UQR.git_user_repo_url, EQ.point")

	if err := db.Table("(?) AS sq", subquery).
		Joins("JOIN questions ON questions.id = sq.question_id").
		Select("sq.user_id, sq.question_id, questions.title AS question_title, sq.git_user_repo_url, sq.score, sq.point, (sq.score / 100 * sq.point) AS weighted_score").
		Find(&questionScores).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to get question scores",
		})
		return
	}

	// Map to organize question scores by user
	userQuestionScores := make(map[uint][]EnhancedQuestionScore)
	for _, qs := range questionScores {
		userQuestionScores[qs.UserID] = append(userQuestionScores[qs.UserID], EnhancedQuestionScore{
			QuestionID:     qs.QuestionID,
			QuestionTitle:  qs.QuestionTitle,
			GitUserRepoURL: qs.GitUserRepoURL,
			Score:          qs.Score,
			WeightedScore:  qs.WeightedScore,
		})
	}

	// Assemble the final leaderboard response with the enhanced structure
	var leaderboardScores []EnhancedLeaderboardScore
	for _, user := range usersWithScores {
		userName := user.UserName
		if !user.IsPublic {
			userName = fmt.Sprintf("User_%d", user.UserID)
		}

		leaderboardScores = append(leaderboardScores, EnhancedLeaderboardScore{
			UserName:       userName,
			TotalScore:     user.TotalScore,
			QuestionScores: userQuestionScores[user.UserID],
		})
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Successfully retrieved exam leaderboard",
		Data: EnhancedGetLeaderboardResponseData{
			Count:  int(totalCount),
			Scores: leaderboardScores,
		},
	})
}
