package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	pb "OJ-API/proto"
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

	// 初始化沙箱調度器
	scheduler := services.GetSandboxScheduler()
	defer scheduler.Close()

	// 創建 gRPC 服務器
	grpcServer := grpc.NewServer()
	pb.RegisterSchedulerServiceServer(grpcServer, scheduler)

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

	// 創建一個多路複用處理器，可以同時處理 HTTP 和 gRPC 請求
	mux := http.NewServeMux()

	// 將 Gin 路由器包裝為 HTTP 處理器
	mux.Handle("/", r)

	// 創建 HTTP 服務器，支持 HTTP/2
	httpServer := &http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// 檢查是否為 gRPC 請求
			if req.ProtoMajor == 2 && strings.HasPrefix(req.Header.Get("Content-Type"), "application/grpc") {
				grpcServer.ServeHTTP(w, req)
			} else {
				// 處理普通 HTTP 請求
				mux.ServeHTTP(w, req)
			}
		}), &http2.Server{}),
	}

	// Signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Received interrupt signal, cleaning up...")
		httpServer.Shutdown(context.Background())
		grpcServer.GracefulStop()
		scheduler.Close()
		os.Exit(0)
	}()

	// 獲取端口並啟動服務器
	port := config.Config("API_PORT")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to listen on port:", err)
	}

	log.Printf("Server (HTTP + gRPC) is running on port %s...", port)
	if err := httpServer.Serve(lis); err != nil && err != http.ErrServerClosed {
		log.Fatal("Failed to start server:", err)
	}
}
