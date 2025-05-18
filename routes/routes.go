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
	// Middleware to handle CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, handlers.ResponseHTTP{
			Success: true,
			Message: "Welcome to the OJ API. Swagger documentation is available at /swagger/index.html",
			Data: map[string]string{
				"swagger_url": "/swagger",
				"api_url":     "/api",
			},
		})
	})

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Swagger documentation
	// Place specific routes before the catch-all route to avoid conflicts
	// Redirect /swagger and /swagger/ to the Swagger documentation index
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	// Catch-all route for Swagger endpoints
	r.GET("/swagger/*any", func(c *gin.Context) {
		// Skip processing if the path is exactly "/swagger/"
		if c.Param("any") == "" || c.Param("any") == "/" {
			c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
			return
		}
		ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.DefaultModelsExpandDepth(0))(c)
	})

	// Routes
	api := r.Group("/api")
	{
		// Admin routes
		api.POST("/admin/user/:id/reset_password", AuthMiddleware(), handlers.ResetUserPassword)
		api.GET("/admin/user", AuthMiddleware(), handlers.GetAllUserInfo)
		api.GET("/admin/user/:id", AuthMiddleware(), handlers.GetUserInfo)
		api.PATCH("/admin/user/:id", AuthMiddleware(), handlers.UpdateUserInfo)

		// Exam routes
		api.POST("/exams", AuthMiddleware(), handlers.CreateExam)
		api.GET("/exams/:id", AuthMiddleware(), handlers.GetExam)
		api.PUT("/exams/:id", AuthMiddleware(), handlers.UpdateExam)
		api.DELETE("/exams/:id", AuthMiddleware(), handlers.DeleteExam)
		api.GET("/exams", handlers.ListExams)
		api.GET("/exams/:id/questions", handlers.GetExamQuestions)
		api.POST("/exams/:id/questions/:question_id", AuthMiddleware(), handlers.AddQuestionToExam)
		api.DELETE("/exams/:id/questions/:question_id", AuthMiddleware(), handlers.RemoveQuestionFromExam)
		api.GET("/exams/:id/score/top", AuthMiddleware(), handlers.GetTopExamScore)
		api.GET("/exams/:id/leaderboard", handlers.GetExamLeaderboard)

		// Sandbox routes
		api.POST("/sandbox", AuthMiddleware(), handlers.PostSandboxCmd)
		api.GET("/sandbox/status", handlers.GetSandboxStatus)

		// Gitea routes
		api.POST("/gitea", handlers.PostGiteaHook)
		api.POST("/gitea/auth", handlers.PostBasicAuthenticationGitea)
		api.POST("/gitea/question/:question_id", AuthMiddleware(), handlers.PostCreateQuestionRepositoryGitea)
		api.GET("/gitea/user", AuthMiddleware(), handlers.GetUserProfileGitea)
		api.POST("/gitea/user/bulk", AuthMiddleware(), handlers.PostBulkCreateUserGitea)
		api.POST("/gitea/user/keys", AuthMiddleware(), handlers.PostCreatePublicKeyGitea)

		// Question routes
		api.GET("/question", handlers.GetQuestionList)
		api.GET("/question/id/:ID", handlers.GetQuestionByID)
		api.GET("/question/user", AuthMiddleware(), handlers.GetUsersQuestions)
		api.GET("/question/user/id/:ID", AuthMiddleware(), handlers.GetUserQuestionByID)
		api.GET("/question/:UQR_ID", AuthMiddleware(), handlers.GetQuestion)
		api.POST("/question", AuthMiddleware(), handlers.AddQuestion)
		api.PATCH("/question/id/:ID", AuthMiddleware(), handlers.PatchQuestion)
		api.DELETE("/question/id/:ID", AuthMiddleware(), handlers.DeleteQuestion)
		api.GET("/question/test_script/:ID", AuthMiddleware(), handlers.GetQuestionTestScript)

		// Score routes
		api.GET("/score", AuthMiddleware(), handlers.GetScoreByRepo)
		api.GET("/score/all", AuthMiddleware(), handlers.GetAllScore)
		api.GET("/score/leaderboard", handlers.GetLeaderboard)
		api.GET("/score/question/:question_id", AuthMiddleware(), handlers.GetScoreByQuestionID)
		api.POST("/score/question/:question_id/rescore", AuthMiddleware(), handlers.ReScoreQuestion)
		api.GET("/score/top", AuthMiddleware(), handlers.GetTopScore)
		api.POST("/score/user/rescore/:question_id", AuthMiddleware(), handlers.ReScoreUserQuestion)
		api.GET("/score/:UQR_ID", AuthMiddleware(), handlers.GetScoreByUQRID)

		// User routes
		api.GET("/user", AuthMiddleware(), handlers.GetUser)
		api.POST("/user/is_public", AuthMiddleware(), handlers.PostUserIsPublic)
		api.POST("/user/change_password", AuthMiddleware(), handlers.ChangeUserPassword)
	}
}
