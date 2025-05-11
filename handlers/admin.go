package handlers

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"net/http"
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
