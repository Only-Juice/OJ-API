package handlers

import (
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
	"net/http"

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
		},
	})
}
