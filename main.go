package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/routes"
	"OJ-API/sandbox"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := database.Connect(); err != nil {
		log.Panic("Can't connect database:", err.Error())
	}
	sandboxCount, err := strconv.Atoi(config.Config("SANDBOX_COUNT"))
	if err != nil {
		log.Panic("Invalid SANDBOX_COUNT config:", err.Error())
	}
	sandbox.SandboxPtr = sandbox.NewSandbox(sandboxCount)
	go sandbox.SandboxPtr.WorkerLoop(ctx)
	defer sandbox.SandboxPtr.Cleanup()

	// Signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Received interrupt signal, cleaning up...")
		cancel()
		sandbox.SandboxPtr.Cleanup()
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
