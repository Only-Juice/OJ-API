package services

import (
	"OJ-API/config"
	pb "OJ-API/proto"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// SandboxClientManager 管理全局的 sandbox gRPC 客戶端
type SandboxClientManager struct {
	client *SandboxClient
	mutex  sync.RWMutex
}

var (
	clientManager *SandboxClientManager
	once          sync.Once
)

// GetSandboxClientManager 獲取全局客戶端管理器實例
func GetSandboxClientManager() *SandboxClientManager {
	once.Do(func() {
		clientManager = &SandboxClientManager{}
		clientManager.initialize()
	})
	return clientManager
}

// initialize 初始化客戶端連接
func (m *SandboxClientManager) initialize() {
	address := config.GetSandboxGRPCAddress()
	client, err := NewSandboxClient(address)
	if err != nil {
		log.Printf("Failed to initialize sandbox client: %v", err)
		return
	}
	
	m.mutex.Lock()
	m.client = client
	m.mutex.Unlock()
	
	log.Printf("Sandbox gRPC client initialized successfully, connected to %s", address)
}

// GetClient 獲取客戶端實例
func (m *SandboxClientManager) GetClient() *SandboxClient {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.client
}

// ReserveJob 添加任務到沙箱隊列
func (m *SandboxClientManager) ReserveJob(repo string, codePath []byte, userQuestionTableID uint64) error {
	client := m.GetClient()
	if client == nil {
		return fmt.Errorf("sandbox client not initialized")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := client.AddJob(ctx, repo, codePath, userQuestionTableID)
	return err
}

// GetStatus 獲取沙箱狀態
func (m *SandboxClientManager) GetStatus() (*pb.SandboxStatusResponse, error) {
	client := m.GetClient()
	if client == nil {
		return nil, fmt.Errorf("sandbox client not initialized")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return client.GetStatus(ctx)
}

// Close 關閉客戶端連接
func (m *SandboxClientManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}
