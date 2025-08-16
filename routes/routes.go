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

func AuthMiddleware(required ...bool) gin.HandlerFunc {
	isRequired := true
	if len(required) > 0 {
		isRequired = required[0]
	}

	return func(c *gin.Context) {
		var jwt string
		var haveAuth bool

		// First, try to get JWT from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			haveAuth = true
			const bearerPrefix = "Bearer "
			if len(authHeader) > len(bearerPrefix) && authHeader[:len(bearerPrefix)] == bearerPrefix {
				jwt = authHeader[len(bearerPrefix):]
			}
		}

		// If no JWT from header, try to get from access_token cookie
		if jwt == "" {
			cookie, err := c.Cookie("access_token")
			if err == nil {
				haveAuth = true
				jwt = cookie
			}
		}

		// If no JWT found and required, return unauthorized
		if haveAuth && isRequired {
			c.JSON(http.StatusUnauthorized, handlers.ResponseHTTP{
				Success: false,
				Message: "Missing Authorization header or access token cookie",
			})
			c.Abort()
			return
		}

		var jwtClaims *utils.JWTClaims
		if !haveAuth {
			jwtClaims = nil
		} else {
			// Validate access token specifically
			var err error
			jwtClaims, err = utils.ValidateAccessToken(jwt)
			if err != nil {
				c.JSON(http.StatusUnauthorized, handlers.ResponseHTTP{
					Success: false,
					Message: "Invalid access token",
				})
				c.Abort()
				return
			}
		}

		ctx := context.WithValue(c.Request.Context(), models.JWTClaimsKey, jwtClaims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func RegisterRoutes(r *gin.Engine) {
	// Enhanced CORS middleware with comprehensive browser compatibility
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		method := c.Request.Method

		// Handle CORS headers - set before any processing
		// For credentials to work, we cannot use wildcard with specific origin
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			// Fallback for requests without Origin header
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			// Note: Cannot set credentials to true with wildcard origin
		}

		// Comprehensive headers for all browsers including Edge, Chrome, Firefox, Safari
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Language, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Origin, Cache-Control, X-Requested-With, Cookie, Set-Cookie, Access-Control-Allow-Origin, Access-Control-Allow-Credentials, Pragma, Expires, Last-Modified, If-Modified-Since, If-None-Match, ETag, Priority, Sec-Fetch-Dest, Sec-Fetch-Mode, Sec-Fetch-Site, Sec-Ch-Ua, Sec-Ch-Ua-Mobile, Sec-Ch-Ua-Platform, User-Agent, Referer")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Set-Cookie, Authorization, Content-Length, Content-Type, Cache-Control, ETag, Last-Modified, Expires")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// Additional headers for browser compatibility
		c.Writer.Header().Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")

		// Handle preflight OPTIONS requests immediately
		if method == "OPTIONS" {
			// Ensure all CORS headers are set for preflight
			c.Writer.Header().Set("Content-Type", "text/plain")
			c.Writer.Header().Set("Content-Length", "0")
			c.AbortWithStatus(http.StatusNoContent)
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
		// Auth routes
		api.POST("/auth/login", handlers.AuthBasic)
		api.POST("/auth/refresh", handlers.RefreshToken)
		api.POST("/auth/logout", handlers.Logout)
		api.GET("/auth/oauth/callback", handlers.OAuthCallback)

		// Admin routes
		api.POST("/admin/:id/user/reset_password", AuthMiddleware(), handlers.ResetUserPassword)
		api.GET("/admin/user", AuthMiddleware(), handlers.GetAllUserInfo)
		api.GET("/admin/:id/user", AuthMiddleware(), handlers.GetUserInfo)
		api.PATCH("/admin/:id/user", AuthMiddleware(), handlers.UpdateUserInfo)

		// Exam routes
		api.POST("/exams/admin", AuthMiddleware(), handlers.CreateExam)
		api.GET("/exams/admin/:id/exam", AuthMiddleware(), handlers.GetExam)
		api.PUT("/exams/admin/:id/exam", AuthMiddleware(), handlers.UpdateExam)
		api.DELETE("/exams/admin/:id/exam", AuthMiddleware(), handlers.DeleteExam)
		api.GET("/exams", handlers.ListExams)
		api.POST("/exams/admin/:id/questions/:question_id/question", AuthMiddleware(), handlers.AddQuestionToExam)
		api.DELETE("/exams/admin/:id/questions/:question_id/question", AuthMiddleware(), handlers.RemoveQuestionFromExam)
		api.GET("/exams/:id/exam", handlers.GetExamInfo)
		api.GET("/exams/:id/leaderboard", handlers.GetExamLeaderboard)
		api.GET("/exams/:id/questions", AuthMiddleware(false), handlers.GetExamQuestions)
		api.GET("/exams/:id/score/top", AuthMiddleware(), handlers.GetTopExamScore)

		// Sandbox routes
		api.POST("/sandbox/admin/sandbox_cmd", AuthMiddleware(), handlers.PostSandboxCmd)
		api.GET("/sandbox/status", handlers.GetSandboxStatus)

		// Gitea routes
		api.POST("/gitea", handlers.PostGiteaHook)
		api.POST("/gitea/auth", handlers.PostBasicAuthenticationGitea)
		api.POST("/gitea/:question_id/question", AuthMiddleware(), handlers.PostCreateQuestionRepositoryGitea)
		api.GET("/gitea/user", AuthMiddleware(), handlers.GetUserProfileGitea)
		api.POST("/gitea/admin/user/bulk", AuthMiddleware(), handlers.PostBulkCreateUserGitea)
		api.POST("/gitea/admin/user/bulk_v2", AuthMiddleware(), handlers.PostBulkCreateUserGiteav2)
		api.POST("/gitea/user/keys", AuthMiddleware(), handlers.PostCreatePublicKeyGitea)
		api.GET("/gitea/user/keys", AuthMiddleware(), handlers.ListMyPublicKeysGitea)
		api.DELETE("/gitea/user/keys", AuthMiddleware(), handlers.DeletePublicKeyGitea)

		// Questions routes
		api.GET("/questions", AuthMiddleware(false), handlers.GetQuestionList)
		api.GET("/questions/:ID/question", handlers.GetQuestionByID)
		api.PATCH("/questions/admin/:ID/question", AuthMiddleware(), handlers.PatchQuestion)
		api.DELETE("/questions/admin/:ID/question", AuthMiddleware(), handlers.DeleteQuestion)
		api.POST("/questions/admin/question", AuthMiddleware(), handlers.AddQuestion)
		api.GET("/questions/user", AuthMiddleware(), handlers.GetUsersQuestions)
		api.GET("/questions/user/:ID/question", AuthMiddleware(), handlers.GetUserQuestionByID)
		api.GET("/questions/uqr/:UQR_ID/question", AuthMiddleware(), handlers.GetQuestion)
		api.GET("/questions/admin/:ID/test_script", AuthMiddleware(), handlers.GetQuestionTestScript)

		// Score routes
		api.GET("/score", AuthMiddleware(), handlers.GetScoreByRepo)
		api.GET("/score/all", AuthMiddleware(), handlers.GetAllScore)
		api.GET("/score/leaderboard", handlers.GetLeaderboard)
		api.GET("/score/:question_id/question", AuthMiddleware(), handlers.GetScoreByQuestionID)
		api.POST("/score/admin/:question_id/question/rescore", AuthMiddleware(), handlers.ReScoreQuestion)
		api.GET("/score/top", AuthMiddleware(), handlers.GetTopScore)
		api.POST("/score/:question_id/question/user_rescore", AuthMiddleware(), handlers.ReScoreUserQuestion)
		api.GET("/score/uqr/:UQR_ID/score", AuthMiddleware(), handlers.GetScoreByUQRID)

		// User routes
		api.GET("/user", AuthMiddleware(), handlers.GetUser)
		api.POST("/user/is_public", AuthMiddleware(), handlers.PostUserIsPublic)
		api.POST("/user/change_password", AuthMiddleware(), handlers.ChangeUserPassword)
		api.POST("/user/forget_password", handlers.ForgetPassword)
		api.GET("/user/reset_password", handlers.ResetPasswordPage)
		api.POST("/user/reset_password", handlers.ResetPassword)
	}
}
