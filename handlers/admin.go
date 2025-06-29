package handlers

import (
	"crypto/md5"
	"crypto/rand"
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
// @Router /api/admin/user/{id}/reset_password [post]
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
// @Router /api/admin/user/{id} [get]
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

// GetAllUserInfo shows all user information
// @Summary Get all user information
// @Description Get all user information
// @Tags admin
// @Accept json
// @Produce json
// @Param			page	query	int		false	"page number of results to return (1-based)"
// @Param			limit	query	int		false	"page size of results. Default is 10."
// @Success      200 {object} ResponseHTTP{data=[]models.User}
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	offset := (page - 1) * limit

	if err := db.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Remove sensitive gitea_token field from all users
	for i := range users {
		users[i].GiteaToken = ""
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    users,
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
// @Router /api/admin/user/{id} [patch]
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

	// Only update fields that are present in the request
	if enable, ok := updateUserInfoDTO["enable"]; ok {
		if enableBool, ok := enable.(bool); ok {
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

	user.GiteaToken = "" // Remove sensitive gitea_token field

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Data:    user,
		Message: "User info updated successfully",
	})
}
