package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"OJ-API/handlers"
	"OJ-API/models"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r)
	return r
}

func TestRootRoute(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.ResponseHTTP
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "Welcome to the OJ API")
}

func TestSwaggerRoutes(t *testing.T) {
	router := setupTestRouter()

	// Test /swagger redirect
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMovedPermanently, w.Code)

	// Test /swagger/ redirect
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/swagger/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMovedPermanently, w.Code)
}

func TestCORSMiddleware(t *testing.T) {
	router := setupTestRouter()

	// Test OPTIONS request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/user", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		header         string
		required       bool
		expectedStatus int
	}{
		{
			name:           "Missing header - required",
			header:         "",
			required:       true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing header - not required",
			header:         "",
			required:       false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid format - required",
			header:         "InvalidToken",
			required:       true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid format - not required",
			header:         "InvalidToken",
			required:       false,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware(tt.required))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAuthMiddlewareWithValidJWT(t *testing.T) {
	// This test would require a valid JWT token
	// You might need to mock the utils.ParseJWT function or create a valid test token
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		claims := c.Request.Context().Value(models.JWTClaimsKey)
		if claims != nil {
			c.JSON(http.StatusOK, gin.H{"message": "authenticated"})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "no claims"})
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	// You would need to generate a valid JWT token here
	// For now, this test demonstrates the structure
	req.Header.Set("Authorization", "Bearer invalid_token")

	r.ServeHTTP(w, req)
	// This will fail with invalid token, but shows the test structure
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
