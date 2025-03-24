package routes

import (
	"context"
	"net/http"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"OJ-API/config"
	_ "OJ-API/docs"
	"OJ-API/handlers"
	"OJ-API/models"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, handlers.ResponseHTTP{
				Success: false,
				Message: "Missing Token",
			})
			c.Abort()
			return
		}
		token = strings.TrimPrefix(token, "token ")
		client, err := gitea.NewClient("http://"+config.Config("GIT_HOST"), gitea.SetToken(token))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		ctx := context.WithValue(c.Request.Context(), models.ClientContextKey, client)
		user, _, err := client.GetMyUserInfo()
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		ctx = context.WithValue(ctx, models.UserContextKey, user)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func RegisterRoutes(r *gin.Engine) {
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Routes
	api := r.Group("/api")
	{
		api.POST("/sandbox", AuthMiddleware(), handlers.PostSandboxCmd)
		api.GET("/sandbox/status", handlers.GetSandboxStatus)
		api.GET("/score", AuthMiddleware(), handlers.GetScoreByRepo)
		api.POST("/gitea", handlers.PostGiteaHook)
		api.POST("/gitea/auth", handlers.PostBasicAuthenticationGitea)
		api.POST("/gitea/user/bulk", AuthMiddleware(), handlers.PostBulkCreateUserGitea)
		api.POST("/gitea/question/:question_id", AuthMiddleware(), handlers.PostCreateQuestionRepositoryGitea)
		api.GET("/question", handlers.GetQuestionList)
		api.GET("/question/user", AuthMiddleware(), handlers.GetUsersQuestions)
		api.GET("/question/:UQR_ID", AuthMiddleware(), handlers.GetQuestion)
		api.GET("/score/:UQR_ID", AuthMiddleware(), handlers.GetScoreByUQRID)
	}
}
