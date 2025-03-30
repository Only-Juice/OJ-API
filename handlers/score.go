package handlers

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
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
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		503
//	@Router			/api/score [get]
//	@Security		BearerAuth
func GetScoreByRepo(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	owner, err := url.QueryUnescape(c.Query("owner"))
	if err != nil || owner == "" {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "Invalid or missing owner parameter",
		})
		return
	}
	if jwtClaims.Username != owner {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}
	repo, err := url.QueryUnescape(c.Query("repo"))
	if err != nil || repo == "" {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "Invalid or missing repo parameter",
		})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	repoURL := fmt.Sprintf("%s/%s", owner, repo)
	var totalCount int64
	if err := db.Model(&models.UserQuestionTable{}).
		Joins("UQR").
		Where("git_user_repo_url = ?", repoURL).
		Count(&totalCount).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to count scores",
		})
		return
	}

	var _scores []models.UserQuestionTable
	if err := db.Model(&models.UserQuestionTable{}).
		Joins("UQR").
		Where("git_user_repo_url = ?", repoURL).
		Offset(offset).
		Limit(limit).
		Find(&_scores).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, ResponseHTTP{
				Success: false,
				Message: "Score not found",
			})
			return
		}
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to get score by repo",
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
	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Successfully retrieved scores by repo",
		Data: GetScoreResponseData{
			Scores:      scores,
			ScoresCount: int(totalCount),
		},
	})
}

// GetScore by UQR ID
//
//	@Summary		Get a score by UQR ID
//	@Description	Get a score by UQR ID
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Param			UQR_ID	path	int	true	"UQR ID"
//	@Param			page		query	int		false	"page number of results to return (1-based)"
//	@Param			limit		query	int		false	"page size of results. Default is 10."
//	@Success		200		{object}	ResponseHTTP{data=Score}
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		503
//	@Router			/api/score/{UQR_ID} [get]
//	@Security		BearerAuth
func GetScoreByUQRID(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	UQRID := c.Param("UQR_ID")
	if UQRID == "" {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "UQR ID is required",
		})
		return
	}

	UQR := models.UserQuestionRelation{}
	if err := db.Where("id = ?", UQRID).First(&UQR).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, ResponseHTTP{
				Success: false,
				Message: "UQR ID not found",
			})
			return
		}
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to get UQR by ID",
		})
		return
	}
	if UQR.UserID != jwtClaims.UserID {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var totalCount int64
	if err := db.Model(&models.UserQuestionTable{}).
		Where("UQR_ID = ?", UQRID).
		Count(&totalCount).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to count scores",
		})
		return
	}

	var _scores []models.UserQuestionTable
	if err := db.Model(&models.UserQuestionTable{}).
		Where("UQR_ID = ?", UQRID).
		Offset(offset).
		Limit(limit).
		Find(&_scores).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, ResponseHTTP{
				Success: false,
				Message: "Score not found",
			})
			return
		}
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to get score by UQR ID",
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
	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Successfully retrieved scores by UQR ID",
		Data: GetScoreResponseData{
			Scores:      scores,
			ScoresCount: int(totalCount),
		},
	})
}

// GetScoreByQuestionID is a function to get a score by question ID
//
//	@Summary		Get a score by question ID
//	@Description	Get a score by question ID
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Param			question_id	path	int	true	"question ID"
//	@Param			page			query	int		false	"page number of results to return (1-based)"
//	@Param			limit			query	int		false	"page size of results. Default is 10."
//	@Success		200			{object}	ResponseHTTP{data=Score}
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		503
//	@Router			/api/score/question/{question_id} [get]
//	@Security		BearerAuth
func GetScoreByQuestionID(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	questionID := c.Param("question_id")
	if questionID == "" {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "Question ID is required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var totalCount int64
	if err := db.Model(&models.UserQuestionRelation{}).
		Where("question_id = ? AND user_id = ?", questionID, jwtClaims.UserID).
		Count(&totalCount).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to count scores",
		})
		return
	}

	var _scores []models.UserQuestionTable
	if err := db.Model(&models.UserQuestionTable{}).
		Joins("UQR").
		Where("question_id = ? AND user_id = ?", questionID, jwtClaims.UserID).
		Offset(offset).
		Limit(limit).
		Find(&_scores).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, ResponseHTTP{
				Success: false,
				Message: "Score not found",
			})
			return
		}
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to get score by question ID",
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
	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Successfully retrieved scores by question ID",
		Data: GetScoreResponseData{
			Scores:      scores,
			ScoresCount: int(totalCount),
		},
	})
}
