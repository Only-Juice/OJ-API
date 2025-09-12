package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
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
	"google.golang.org/grpc/credentials"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	pb "OJ-API/proto"
	"OJ-API/routes"
	"OJ-API/services"
	"OJ-API/utils"
)

// @title			OJ-PoC API
// @version		1.0
// @description	This is a simple OJ-PoC API server.
// @BasePath		/
// @SecurityDefinitions.apikey BearerAuth
// @In header
// @Name Authorization
func main() {
	// 初始化日誌
	utils.InitLog()
	utils.SetServerSource("api-server")

	// 檢查 TLS 憑證文件
	certFile := config.Config("TLS_CERT_FILE")
	keyFile := config.Config("TLS_KEY_FILE")
	useTLS := false

	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			useTLS = true
			utils.Info("TLS certificates found, enabling HTTPS")
		} else {
			utils.Warn("cert.pem found but key.pem missing, using HTTP")
		}
	} else {
		utils.Info("No TLS certificates found, using HTTP")
	}

	decodedKey, err := base64.StdEncoding.DecodeString(config.Config("ENCRYPTION_KEY"))
	if err != nil {
		utils.Fatal("Invalid ENCRYPTION_KEY config:", err.Error())
	}
	if len(decodedKey) != 16 && len(decodedKey) != 24 && len(decodedKey) != 32 {
		utils.Fatal("Invalid ENCRYPTION_KEY length:", len(decodedKey))
	}

	if err := database.Connect(); err != nil {
		utils.Fatal("Can't connect database:", err.Error())
	}

	// 初始化沙箱調度器
	scheduler := services.GetSandboxScheduler()
	defer scheduler.Close()

	// 創建 gRPC 服務器
	var grpcServer *grpc.Server
	if useTLS {
		// 載入 TLS 憑證
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			utils.Fatal("Failed to load TLS certificates:", err)
		}

		// 創建 TLS 憑證
		creds := credentials.NewServerTLSFromCert(&cert)
		grpcServer = grpc.NewServer(grpc.Creds(creds))
	} else {
		grpcServer = grpc.NewServer()
	}
	pb.RegisterSchedulerServiceServer(grpcServer, scheduler)

	// Database migrations
	models := []interface{}{
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
	}

	for _, m := range models {
		if err := database.DBConn.AutoMigrate(m); err != nil {
			utils.Errorf("AutoMigrate %T failed: %v", m, err)
		}
	}

	// Initialize Gin router
	r := gin.Default()
	routes.RegisterRoutes(r)

	// 創建一個多路複用處理器，可以同時處理 HTTP 和 gRPC 請求
	mux := http.NewServeMux()

	// 將 Gin 路由器包裝為 HTTP 處理器
	mux.Handle("/", r)

	// 創建 HTTP 服務器
	var httpServer *http.Server

	if useTLS {
		// HTTPS 模式：直接處理請求，因為 TLS 終止在此層
		httpServer = &http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/health" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
					return
				}

				if req.ProtoMajor == 2 && strings.HasPrefix(req.Header.Get("Content-Type"), "application/grpc") {
					grpcServer.ServeHTTP(w, req)
				} else {
					mux.ServeHTTP(w, req)
				}
			}),
		}
	} else {
		// HTTP 模式：使用 h2c 支持 HTTP/2
		httpServer = &http.Server{
			Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/health" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
					return
				}

				if req.ProtoMajor == 2 && strings.HasPrefix(req.Header.Get("Content-Type"), "application/grpc") {
					grpcServer.ServeHTTP(w, req)
				} else {
					mux.ServeHTTP(w, req)
				}
			}), &http2.Server{}),
		}
	}

	// Signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		utils.Info("Received interrupt signal, cleaning up...")
		httpServer.Shutdown(context.Background())
		grpcServer.GracefulStop()
		scheduler.Close()
		os.Exit(0)
	}()

	// 獲取端口並啟動服務器
	port := config.Config("API_PORT")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		utils.Fatal("Failed to listen on port:", err)
	}

	// Gitea Check
	go services.GiteaCheckInit()

	if useTLS {
		utils.Infof("Server (HTTPS + gRPC) is running on port %s...", port)
		if err := httpServer.ServeTLS(lis, certFile, keyFile); err != nil && err != http.ErrServerClosed {
			utils.Fatal("Failed to start HTTPS server:", err)
		}
	} else {
		utils.Infof("Server (HTTP + gRPC) is running on port %s...", port)
		if err := httpServer.Serve(lis); err != nil && err != http.ErrServerClosed {
			utils.Fatal("Failed to start server:", err)
		}
	}
}
