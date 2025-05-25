package handlers

import (
	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
	"net/http"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"
)

type IsPublicRequest struct {
	IsPublic bool `json:"is_public"`
}

// Post User is_public setting
// @Summary Set user is_public
// @Description Update the user's is_public setting
// @Tags User
// @Accept json
// @Produce json
// @Param is_public body handlers.IsPublicRequest true "is_public"
// @Success 200 {object} ResponseHTTP{data=models.User}
// @Failure 400 {object} ResponseHTTP{}
// @Failure 401 {object} ResponseHTTP{}
// @Failure 404 {object} ResponseHTTP{}
// @Failure 503 {object} ResponseHTTP{}
// @Router /api/user/is_public [post]
// @Security BearerAuth
func PostUserIsPublic(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	// Find the user by ID
	var user models.User
	if err := db.First(&user, jwtClaims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Bind the request body to IsPublicRequest
	var req IsPublicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Update the user's is_public field
	user.IsPublic = req.IsPublic
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to update user",
		})
		return
	}

	// Respond with success
	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "User is_public updated successfully",
		Data: models.User{
			ID:       user.ID,
			UserName: user.UserName,
			Enable:   user.Enable,
			Email:    user.Email,
			IsPublic: user.IsPublic,
		},
	})
}

type GetUserData struct {
	ID       uint   `json:"id"`
	UserName string `json:"user_name"`
	Enable   bool   `json:"enable"`
	Email    string `json:"email"`
	IsPublic bool   `json:"is_public"`
	IsAdmin  bool   `json:"is_admin"`
}

// Get User Info
// @Summary Get user info
// @Description Get user info by ID
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} ResponseHTTP{data=GetUserData}
// @Failure 400 {object} ResponseHTTP{}
// @Failure 401 {object} ResponseHTTP{}
// @Failure 404 {object} ResponseHTTP{}
// @Failure 503 {object} ResponseHTTP{}
// @Router /api/user [get]
// @Security BearerAuth
func GetUser(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	// Find the user by ID
	var user models.User
	if err := db.First(&user, jwtClaims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Respond with user info
	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "User info retrieved successfully",
		Data: GetUserData{
			ID:       user.ID,
			UserName: user.UserName,
			Enable:   user.Enable,
			Email:    user.Email,
			IsPublic: user.IsPublic,
			IsAdmin:  user.IsAdmin,
		},
	})
}

type ChangeUserPasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// Change User Password
// @Summary Change user password
// @Description Change user password
// @Tags User
// @Accept json
// @Produce json
// @Param request body ChangeUserPasswordRequest true "ChangeUserPasswordRequest"
// @Success 200 {object} ResponseHTTP{}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/user/change_password [post]
func ChangeUserPassword(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	request := ChangeUserPasswordRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	client_check, err := gitea.NewClient("http://"+config.Config("GIT_HOST"),
		gitea.SetBasicAuth(jwtClaims.Username, request.OldPassword),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	_, _, err = client_check.GetMyUserInfo()
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to authenticate with Gitea",
		})
		return
	}

	// Find Admin user
	var adminUser models.User
	if err := db.First(&adminUser, models.User{
		IsAdmin: true,
	}).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}
	// Update password
	token, err := utils.GetToken(adminUser.ID)
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
	_, err = client.AdminEditUser(jwtClaims.Username, gitea.EditUserOption{
		LoginName: jwtClaims.Username,
		Password:  request.NewPassword,
	})
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "Password changed successfully",
	})
}
