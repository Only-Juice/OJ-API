package routes

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "OJ-API/docs"
	"OJ-API/handlers"
	"OJ-API/models"
	"OJ-API/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, handlers.ResponseHTTP{
				Success: false,
				Message: "Missing Authorization header",
			})
			c.Abort()
			return
		}

		const bearerPrefix = "Bearer "
		if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
			c.JSON(http.StatusBadRequest, handlers.ResponseHTTP{
				Success: false,
				Message: "Invalid Authorization header format",
			})
			c.Abort()
			return
		}

		jwt := authHeader[len(bearerPrefix):]

		jwtClaims, err := utils.ParseJWT(jwt)
		if err != nil {
			c.JSON(http.StatusUnauthorized, handlers.ResponseHTTP{
				Success: false,
				Message: "Invalid JWT",
			})
			c.Abort()
			return
		}

		ctx := context.WithValue(c.Request.Context(), models.JWTClaimsKey, jwtClaims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func RegisterRoutes(r *gin.Engine) {
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.DefaultModelsExpandDepth(0)))

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
		api.GET("/question/id/:ID", handlers.GetQuestionByID)
		api.GET("/question/user/id/:ID", AuthMiddleware(), handlers.GetUserQuestionByID)
		api.GET("/score/:UQR_ID", AuthMiddleware(), handlers.GetScoreByUQRID)
		api.GET("/gitea/user", AuthMiddleware(), handlers.GetUserProfileGitea)
	}
}
