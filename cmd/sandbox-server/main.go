package main

import (
	"OJ-API/config"
	"OJ-API/database"
	pb "OJ-API/proto"
	"OJ-API/sandbox"
	"OJ-API/services"
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	// 加載環境變數
	if err := godotenv.Load(".env.local"); err != nil {
		log.Println("No .env.local file found")
	}

	// 初始化數據庫連接
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 創建沙箱實例
	sandboxCount := runtime.NumCPU() // 或者從配置文件讀取
	if countStr := config.Config("SANDBOX_COUNT"); countStr != "" {
		// 可以從環境變數設置沙箱數量
	}

	sandboxInstance := sandbox.NewSandbox(sandboxCount)
	defer sandboxInstance.Cleanup()

	// 啟動工作循環
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sandboxInstance.WorkerLoop(ctx)

	// 創建 gRPC 服務器
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	grpcServer := grpc.NewServer()
	sandboxServer := services.NewSandboxServer(sandboxInstance)
	pb.RegisterSandboxServiceServer(grpcServer, sandboxServer)

	log.Println("Sandbox gRPC server is running on port 50051...")

	// 優雅關機處理
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down gRPC server...")
		cancel() // 停止工作循環
		grpcServer.GracefulStop()
	}()

	// 啟動服務器
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
