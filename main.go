package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/routes"
	"OJ-API/services"
)

// @title			OJ-PoC API
// @version		1.0
// @description	This is a simple OJ-PoC API server.
// @BasePath		/
// @SecurityDefinitions.apikey BearerAuth
// @In header
// @Name Authorization
func main() {
	decodedKey, err := base64.StdEncoding.DecodeString(config.Config("ENCRYPTION_KEY"))
	if err != nil {
		log.Panic("Invalid ENCRYPTION_KEY config:", err.Error())
	}
	if len(decodedKey) != 16 && len(decodedKey) != 24 && len(decodedKey) != 32 {
		log.Panic("Invalid ENCRYPTION_KEY length:", len(decodedKey))
	}

	if err := database.Connect(); err != nil {
		log.Panic("Can't connect database:", err.Error())
	}
	
	// 初始化 gRPC 客戶端管理器
	clientManager := services.GetSandboxClientManager()
	defer clientManager.Close()

	// Signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Received interrupt signal, cleaning up...")
		clientManager.Close()
		os.Exit(0)
	}()

	// Database migrations
	database.DBConn.AutoMigrate(
		&models.User{},
		&models.Announcement{},
		&models.Exam{},
		&models.Question{},
		&models.ExamQuestion{},
		&models.QuestionTestScript{},
		&models.Tag{},
		&models.TagAndQuestion{},
		&models.UserQuestionRelation{},
		&models.UserQuestionTable{},
	)

	// Initialize Gin router
	r := gin.Default()
	routes.RegisterRoutes(r)

	// Start the server
	port := config.Config("API_PORT")
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
