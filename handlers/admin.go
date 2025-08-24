package handlers

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
)

type ResetUserPasswordDTO struct {
	Password string `json:"password" binding:"required"`
}

// ResetUserPassword resets the password of a user
// @Summary Reset user password
// @Description Reset the password of a user
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success      200 {object} ResponseHTTP{data=ResetUserPasswordDTO}
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router /api/admin/{id}/user/reset_password [post]
// @Security BearerAuth
func ResetUserPassword(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}
	db := database.DBConn
	id := c.Param("id")
	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}
	if jwtClaims.UserID == uint(userID) {
		c.JSON(http.StatusForbidden, ResponseHTTP{
			Success: false,
			Message: "Cannot reset your own password",
		})
		return
	}
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to generate secure salt",
		})
		return
	}
	saltStr := fmt.Sprintf("%x", salt)

	hasher := md5.New()
	hasher.Write([]byte(time.Now().String() + saltStr))
	passwordHashBytes := hasher.Sum(nil)

	passwordHash := fmt.Sprintf("%x", passwordHashBytes)[:8]

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

	client.AdminEditUser(user.UserName, gitea.EditUserOption{
		LoginName: user.UserName,
		Password:  passwordHash,
	})

	// Send password reset notification email
	if err := utils.SendPasswordResetNotification(user.Email, user.UserName, passwordHash); err != nil {
		// Log error but don't fail the request
		utils.Warnf("Failed to send password reset notification email to %s: %v", user.Email, err)
	}

	// Update user's reset password status
	user.ResetPassword = true
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to update user",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data: ResetUserPasswordDTO{
			Password: passwordHash,
		},
		Message: "Password reset successfully",
	})
}

// GetUserInfo shows the user information
// @Summary Get user information
// @Description Get the user information
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success      200 {object} ResponseHTTP{data=models.User}
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router /api/admin/{id}/user [get]
// @Security BearerAuth
func GetUserInfo(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}
	db := database.DBConn
	id := c.Param("id")
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Remove sensitive gitea_token field
	user.GiteaToken = ""

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    user,
		Message: "User info retrieved successfully",
	})
}

type GetAllUserInfoResponse struct {
	Items      []models.User `json:"items"`
	TotalCount int64         `json:"total_count"`
}

// GetAllUserInfo shows all user information
// @Summary Get all user information
// @Description Get all user information
// @Tags admin
// @Accept json
// @Produce json
// @Param			page	query	int		false	"page number of results to return (1-based)"
// @Param			limit	query	int		false	"page size of results. Default is 10."
// @Success      200 {object} ResponseHTTP{data=GetAllUserInfoResponse}
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router /api/admin/user [get]
// @Security BearerAuth
func GetAllUserInfo(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}
	db := database.DBConn
	var users []models.User
	var total int64
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	offset := (page - 1) * limit

	if err := db.Model(&models.User{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to count users",
		})
		return
	}

	if err := db.Limit(limit).Offset(offset).Order("is_admin DESC, id ASC").Find(&users).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Remove sensitive gitea_token field from all users
	for i := range users {
		users[i].GiteaToken = ""
		users[i].RefreshToken = ""
	}

	response := GetAllUserInfoResponse{
		TotalCount: total,
		Items:      users,
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    response,
		Message: "User info retrieved successfully",
	})
}

type UpdateUserInfoDTO struct {
	Enable   bool `json:"enable"`
	IsPublic bool `json:"is_public"`
}

// UpdateUserInfo updates the user information
// @Summary Update user information
// @Description Update the user information (partially or fully)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body UpdateUserInfoDTO false "User information"
// @Success      200 {object} ResponseHTTP{data=models.User}
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router /api/admin/{id}/user [patch]
// @Security BearerAuth
func UpdateUserInfo(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}
	db := database.DBConn
	id := c.Param("id")
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	var updateUserInfoDTO map[string]interface{}
	if err := c.ShouldBindJSON(&updateUserInfoDTO); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Check if admin is trying to modify their own account
	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	// Only update fields that are present in the request
	if enable, ok := updateUserInfoDTO["enable"]; ok {
		if enableBool, ok := enable.(bool); ok {
			if jwtClaims.UserID == uint(userID) && user.Enable != enableBool {
				c.JSON(http.StatusForbidden, ResponseHTTP{
					Success: false,
					Message: "Cannot modify your own account enable status",
				})
				return
			}
			user.Enable = enableBool
		}
	}

	if isPublic, ok := updateUserInfoDTO["is_public"]; ok {
		if isPublicBool, ok := isPublic.(bool); ok {
			user.IsPublic = isPublicBool
		}
	}

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to update user info",
		})
		return
	}

	// Remove sensitive gitea_token and refresh_token field
	user.GiteaToken = ""
	user.RefreshToken = ""

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    user,
		Message: "User info updated successfully",
	})
}

// Export Question Score
// @Summary Export question score
// @Description Export question score to CSV or XLSX
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Question ID"
// @Param format query string false "Export format: csv or xlsx" default(csv)
// @Success 200 {file} application/csv
// @Success 200 {file} application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/admin/questions/{id}/export [get]
// @Security BearerAuth
func ExportQuestionScore(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}
	db := database.DBConn
	id := c.Param("id")
	format := c.DefaultQuery("format", "csv")

	// Validate format parameter
	if format != "csv" && format != "xlsx" {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid format. Supported formats: csv, xlsx",
		})
		return
	}

	var question models.Question
	if err := db.First(&question, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	// Fetch question scores with earliest submit time for highest score
	var scores []utils.ExportQuestionScoreResponse
	if err := db.Table("user_question_relations UQR").
		Select("U.user_name as user_name, UQR.git_user_repo_url as git_user_repo_url, COALESCE(MAX(UQT.score), 0) AS score, MIN(CASE WHEN UQT.score = (SELECT MAX(score) FROM user_question_tables WHERE uqr_id = UQR.id) THEN UQT.created_at END) AS earliest_best_submit_time").
		Where("UQR.question_id = ?", question.ID).
		Joins("JOIN users U ON U.id = UQR.user_id").
		Joins("LEFT JOIN user_question_tables UQT ON UQT.uqr_id = UQR.id").
		Group("U.user_name, UQR.git_user_repo_url").
		Find(&scores).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to fetch question scores",
		})
		return
	}

	if format == "csv" {
		// Generate CSV
		var csvData bytes.Buffer
		// Add UTF-8 BOM for proper encoding
		csvData.Write([]byte{0xEF, 0xBB, 0xBF})
		writer := csv.NewWriter(&csvData)

		// Write CSV header
		writer.Write([]string{"User Name", "Git User Repo URL", "Score", "Earliest Best Submit Time"})

		// Write CSV rows
		for _, score := range scores {
			writer.Write([]string{
				score.UserName,
				score.GitUserRepoURL,
				strconv.FormatFloat(score.Score, 'f', 2, 64),
				score.EarliestBestSubmitTime.Format("2006-01-02 15:04:05"),
			})
		}

		// Flush the writer to ensure all data is written to the buffer
		writer.Flush()

		// Set response headers for CSV
		c.Header("Content-Type", "application/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"question_%d_%s.csv\"", question.ID, time.Now().Format("20060102_150405")))
		c.String(http.StatusOK, csvData.String())
	} else {
		// Generate XLSX
		if err := utils.ExportQuestionScoreToXLSX(c, question.ID, scores); err != nil {
			c.JSON(http.StatusInternalServerError, ResponseHTTP{
				Success: false,
				Message: "Failed to generate XLSX file",
			})
			return
		}
	}
}
