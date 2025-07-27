package services

import (
	pb "OJ-API/proto"
	"sync"
)

// SandboxClientManager 管理全局的 sandbox 調度
type SandboxClientManager struct {
	scheduler *SandboxScheduler
}

var (
	clientManager *SandboxClientManager
	once          sync.Once
)

// GetSandboxClientManager 獲取全局客戶端管理器實例
func GetSandboxClientManager() *SandboxClientManager {
	once.Do(func() {
		clientManager = &SandboxClientManager{
			scheduler: GetSandboxScheduler(),
		}
	})
	return clientManager
}

// ReserveJob 添加任務到沙箱隊列
func (m *SandboxClientManager) ReserveJob(parentGitFullName string, gitRepoURL string, gitFullName string, gitAfterHash string, gitUsername string, gitToken string, userQuestionTableID uint64) error {
	return m.scheduler.ReserveJob(parentGitFullName, gitRepoURL, gitFullName, gitAfterHash, gitUsername, gitToken, userQuestionTableID)
}

// GetStatus 獲取沙箱狀態
func (m *SandboxClientManager) GetStatus() (*pb.SandboxStatusResponse, error) {
	return m.scheduler.GetGlobalStatus(), nil
}

// Close 關閉客戶端連接
func (m *SandboxClientManager) Close() error {
	m.scheduler.Close()
	return nil
}
