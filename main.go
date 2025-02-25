package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/routes"
	"OJ-API/sandbox"
)

// @title			OJ-PoC API
// @version		1.0
// @description	This is a simple OJ-PoC API server.
// @BasePath		/
func main() {
	if err := database.Connect(); err != nil {
		log.Panic("Can't connect database:", err.Error())
	}
	sandbox.SandboxPtr = sandbox.NewSandbox(10)
	defer sandbox.SandboxPtr.Cleanup()

	// 設置信號處理
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Received interrupt signal, cleaning up...")
		sandbox.SandboxPtr.Cleanup()
		os.Exit(0)
	}()

	sandbox.SandboxPtr.RunShellCommandByRepo("user_name/repo_name", nil)

	database.DBConn.AutoMigrate(&models.Sandbox{})
	database.DBConn.AutoMigrate(&models.Score{})

	r := routes.New()
	log.Fatal(http.ListenAndServe(":3002", r))
}
