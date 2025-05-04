package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

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
