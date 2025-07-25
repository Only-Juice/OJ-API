package main

import (
	"context"
	"log"
	"time"

	"OJ-API/services"
)

func main() {
	log.Println("Testing Sandbox gRPC Client...")

	// 創建客戶端
	client, err := services.NewSandboxClient("localhost:50051")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 測試健康檢查
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := client.HealthCheck(ctx)
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}

	log.Printf("Sandbox Status: Available=%d, Waiting=%d, Processing=%d, Total=%d",
		status.AvailableCount, status.WaitingCount, status.ProcessingCount, status.TotalCount)

	log.Println("gRPC client test completed successfully!")
}
