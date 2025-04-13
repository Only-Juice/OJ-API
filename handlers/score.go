package handlers

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/sandbox"
	"OJ-API/utils"
)

type Score struct {
	Score     float64   `json:"score" example:"100" validate:"required"`
	Message   string    `json:"message" example:"Scored successfully" validate:"required"`
	JudgeTime time.Time `json:"judge_time" example:"2006-01-02T15:04:05Z07:00" time_format:"RFC3339" validate:"required"`
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
		Order("judge_time DESC").
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
	if err := db.Where("id = ?", UQRID).Limit(1).Find(&UQR).Error; err != nil {
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
		Order("judge_time DESC").
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
	if err := db.Model(&models.UserQuestionTable{}).
		Joins("UQR").
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
		Order("judge_time DESC").
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
	if len(scores) == 0 {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "No scores found for this question ID",
		})
		return
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

// ReScoreUserQuestion is a function to re-score a specific user's question by question ID
//
//	@Summary		Re-score a specific user's question
//	@Description	Re-score a specific user's question by question ID
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Param			question_id	path	int	true	"question ID"
//	@Success		200		{object}	ResponseHTTP{}
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		503
//	@Router			/api/score/user/rescore/{question_id} [post]
//	@Security		BearerAuth
func ReScoreUserQuestion(c *gin.Context) {
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

	var question models.Question
	if err := db.Where("id = ?", questionID).Limit(1).Find(&question).Error; err != nil {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	var uqr models.UserQuestionRelation
	if err := db.Model(&models.UserQuestionRelation{}).
		Where("question_id = ? AND user_id = ?", questionID, jwtClaims.UserID).
		Limit(1).Find(&uqr).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to re-score the question",
		})
		return
	}

	newScore := models.UserQuestionTable{
		UQR:       uqr,
		Score:     -3,
		JudgeTime: time.Now().UTC(),
		Message:   "Waiting for judging...",
	}
	if err := db.Create(&newScore).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to create new score entry",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Re-scoring the question",
	})

	go func() {
		codePath := fmt.Sprintf("%s/%s", config.Config("REPO_FOLDER"), uqr.GitUserRepoURL+"/"+uuid.New().String())
		_, err := git.PlainClone(codePath, false, &git.CloneOptions{
			URL:      "http://" + config.Config("GIT_HOST") + "/" + uqr.GitUserRepoURL,
			Progress: os.Stdout,
		})
		if err != nil {
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: "Failed to clone repository",
			})
			return
		}
		os.Chmod(codePath, 0777) // Need to confirm if this is necessary

		defer os.RemoveAll(codePath)

		sandbox.SandboxPtr.RunShellCommandByRepo(question.GitRepoURL, []byte(codePath), newScore)
	}()
}

type TopScore struct {
	QuestionID     int       `json:"question_id" example:"1" validate:"required"`
	GitUserRepoURL string    `json:"git_user_repo_url" example:"owner/repo" validate:"required"`
	Score          float64   `json:"score" example:"100" validate:"required"`
	Message        string    `json:"message" example:"Scored successfully" validate:"required"`
	JudgeTime      time.Time `json:"judge_time" example:"2006-01-02T15:04:05Z07:00" time_format:"RFC3339" validate:"required"`
}

type GetTopScoreResponseData struct {
	ScoresCount int        `json:"scores_count" validate:"required"`
	Scores      []TopScore `json:"scores" validate:"required"`
}

// Get the top score of each question for user
//
//	@Summary		Get the top score of each question for user
//	@Description	Get the top score of each question for user
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Param			page	query	int	false	"page number of results to return (1-based)"
//	@Param			limit	query	int	false	"page size of results. Default is 10."
//	@Success		200	{object}	ResponseHTTP{data=GetTopScoreResponseData}
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		503
//	@Router			/api/score/top [get]
//	@Security		BearerAuth
func GetTopScore(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var totalCount int64

	subQuery := db.Model(&models.UserQuestionTable{}).
		Select("DISTINCT question_id").
		Joins("JOIN user_question_relations UQR ON user_question_tables.uqr_id = UQR.id").
		Where("UQR.user_id = ?", jwtClaims.UserID)

	if err := db.Table("(?) AS sub", subQuery).
		Count(&totalCount).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to count scores",
		})
		return
	}

	var scores []TopScore
	if err := db.Model(&models.UserQuestionTable{}).
		Joins("UQR").
		Select("DISTINCT ON (question_id) question_id, git_user_repo_url, score, message, judge_time").
		Where("user_id = ?", jwtClaims.UserID).
		Order("question_id, score DESC").
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
		Data: GetTopScoreResponseData{
			Scores:      scores,
			ScoresCount: int(totalCount),
		},
	})
}

// ReScoreQuestion is a function to re-score a specific question by question ID
//
//	@Summary		Re-score a specific question
//	@Description	Re-score a specific question by question ID
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Param			question_id	path	int	true	"question ID"
//	@Success		200		{object}	ResponseHTTP{}
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		503
//	@Router			/api/score/question/{question_id}/rescore [post]
//	@Security		BearerAuth
func ReScoreQuestion(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	questionID := c.Param("question_id")
	if questionID == "" {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "Question ID is required",
		})
		return
	}

	var question models.Question
	if err := db.Where("id = ?", questionID).Limit(1).Find(&question).Error; err != nil {
		c.JSON(404, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	var uqr []models.UserQuestionRelation
	if err := db.Model(&models.UserQuestionRelation{}).
		Where("question_id = ? AND user_id = ?", questionID, jwtClaims.UserID).
		Limit(1).Find(&uqr).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to re-score the question",
		})
		return
	}

	newScores := []models.UserQuestionTable{}
	for _, u := range uqr {
		newScores = append(newScores, models.UserQuestionTable{
			UQR:       u,
			Score:     -3,
			JudgeTime: time.Now().UTC(),
			Message:   "Waiting for judging...",
		})
	}

	if err := db.Create(&newScores).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to create new score entry",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Re-scoring the question",
	})

	go func() {
		var wg sync.WaitGroup

		for i, u := range uqr {
			codePath := fmt.Sprintf("%s/%s", config.Config("REPO_FOLDER"), u.GitUserRepoURL+"/"+uuid.New().String())
			_, err := git.PlainClone(codePath, false, &git.CloneOptions{
				URL:      "http://" + config.Config("GIT_HOST") + "/" + u.GitUserRepoURL,
				Progress: os.Stdout,
			})
			if err != nil {
				db.Model(&newScores[i]).Updates(models.UserQuestionTable{
					Score:   -2,
					Message: "Failed to clone repository",
				})
				return
			}
			os.Chmod(codePath, 0777) // Need to confirm if this is necessary

			defer os.RemoveAll(codePath)

			wg.Add(1)
			go func(i int, codePath string) {
				defer wg.Done()
				sandbox.SandboxPtr.RunShellCommandByRepo(question.GitRepoURL, []byte(codePath), newScores[i])
			}(i, codePath)
		}

		wg.Wait()
	}()
}

// GetAllScore is a function to get all scores for the user
//
//	@Summary		Get all scores for the user
//	@Description	Get all scores for the user
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Param			page	query	int	false	"page number of results to return (1-based)"
//	@Param			limit	query	int	false	"page size of results. Default is 10."
//	@Success		200	{object}	ResponseHTTP{data=GetTopScoreResponseData}
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		503
//	@Router			/api/score/all [get]
//	@Security		BearerAuth
func GetAllScore(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var totalCount int64

	subQuery := db.Model(&models.UserQuestionTable{}).
		Joins("JOIN user_question_relations UQR ON user_question_tables.uqr_id = UQR.id").
		Where("UQR.user_id = ?", jwtClaims.UserID)

	if err := db.Table("(?) AS sub", subQuery).
		Count(&totalCount).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to count scores",
		})
		return
	}

	var scores []TopScore
	if err := db.Model(&models.UserQuestionTable{}).
		Joins("UQR").
		Select("question_id, git_user_repo_url, score, message, judge_time").
		Where("user_id = ?", jwtClaims.UserID).
		Order("question_id, judge_time DESC").
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
			Message: "Failed to get all scores",
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
		Message: "Successfully retrieved all scores",
		Data: GetTopScoreResponseData{
			Scores:      scores,
			ScoresCount: int(totalCount),
		},
	})
}

type LeaderboardScore struct {
	UserName string  `json:"user_name" example:"owner" validate:"required"`
	Score    float64 `json:"score" example:"100" validate:"required"`
}

type GetLeaderboardResponseData struct {
	Count  int                `json:"count" validate:"required"`
	Scores []LeaderboardScore `json:"scores" validate:"required"`
}

// GetLeaderboard is a function to get the leaderboard
//
//	@Summary		Get the leaderboard
//	@Description	Get the leaderboard
//	@Tags			Score
//	@Accept			json
//	@Produce		json
//	@Param			page	query	int	false	"page number of results to return (1-based)"
//	@Param			limit	query	int	false	"page size of results. Default is 10."
//	@Success		200	{object}	ResponseHTTP{data=GetLeaderboardResponseData}
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		503
//	@Router			/api/score/leaderboard [get]
func GetLeaderboard(c *gin.Context) {
	db := database.DBConn

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var totalCount int64
	if err := db.Table("(SELECT UQR.user_id AS user_id, MAX(score) AS max_score, UQR.question_id " +
		"FROM user_question_tables " +
		"JOIN user_question_relations UQR ON user_question_tables.uqr_id = UQR.id " +
		"GROUP BY UQR.user_id, UQR.question_id) AS subquery").
		Select("user_id, SUM(max_score) AS score").
		Group("user_id").
		Count(&totalCount).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to count scores",
		})
		return
	}
	var scores []LeaderboardScore
	if err := db.Table("(SELECT UQR.user_id AS user_id, MAX(score) AS max_score, UQR.question_id " +
		"FROM user_question_tables " +
		"JOIN user_question_relations UQR ON user_question_tables.uqr_id = UQR.id " +
		"GROUP BY UQR.user_id, UQR.question_id) AS subquery").
		Joins("JOIN users ON users.id = subquery.user_id").
		Select("CASE WHEN users.is_public THEN users.user_name ELSE CONCAT('User_', users.id) END AS user_name, SUM(max_score) AS score").
		Group("users.user_name, users.is_public, users.id").
		Order("score DESC").
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
			Message: "Failed to get leaderboard",
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
		Message: "Successfully retrieved leaderboard",
		Data: GetLeaderboardResponseData{
			Count:  int(totalCount),
			Scores: scores,
		},
	})
}
