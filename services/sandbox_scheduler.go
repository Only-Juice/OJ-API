package services

import (
	pb "OJ-API/proto"
	"OJ-API/utils"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// SandboxInstance 表示一個沙箱實例
type SandboxInstance struct {
	ID       string
	Capacity int32
	Status   *pb.SandboxStatusResponse
	LastSeen time.Time
	Active   bool
	Stream   pb.SchedulerService_SandboxStreamServer // 雙向流連接
	JobChan  chan *pb.AddJobRequest                  // 任務通道
}

// SandboxScheduler 管理多個沙箱實例的調度
type SandboxScheduler struct {
	pb.UnimplementedSchedulerServiceServer
	instances map[string]*SandboxInstance
	mutex     sync.RWMutex
}

var (
	globalScheduler *SandboxScheduler
	schedulerOnce   sync.Once
)

// GetSandboxScheduler 獲取全局調度器實例
func GetSandboxScheduler() *SandboxScheduler {
	schedulerOnce.Do(func() {
		globalScheduler = &SandboxScheduler{
			instances: make(map[string]*SandboxInstance),
		}
		// 啟動清理 goroutine
		go globalScheduler.cleanupInactiveInstances()
	})
	return globalScheduler
}

// SandboxStream 處理沙箱雙向流連接
func (s *SandboxScheduler) SandboxStream(stream pb.SchedulerService_SandboxStreamServer) error {
	var instance *SandboxInstance
	var sandboxID string

	defer func() {
		if instance != nil {
			s.mutex.Lock()
			instance.Active = false
			close(instance.JobChan)
			delete(s.instances, sandboxID)
			s.mutex.Unlock()
			utils.Infof("Sandbox %s disconnected", sandboxID)
		}
	}()

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			utils.Errorf("Stream receive error: %v", err)
			break
		}

		switch msgType := msg.MessageType.(type) {
		case *pb.SandboxMessage_Connect:
			// 處理沙箱連接
			connectReq := msgType.Connect
			sandboxID = connectReq.SandboxId

			instance = &SandboxInstance{
				ID:       sandboxID,
				Capacity: connectReq.Capacity,
				LastSeen: time.Now(),
				Active:   true,
				Stream:   stream,
				JobChan:  make(chan *pb.AddJobRequest, 100),
			}

			s.mutex.Lock()
			s.instances[sandboxID] = instance
			s.mutex.Unlock()

			// 發送連接響應
			response := &pb.SchedulerMessage{
				SandboxId: sandboxID,
				MessageType: &pb.SchedulerMessage_ConnectResponse{
					ConnectResponse: &pb.RegisterSandboxResponse{
						Success: true,
						Message: "Connected successfully",
					},
				},
			}

			if err := stream.Send(response); err != nil {
				utils.Errorf("Failed to send connect response: %v", err)
				return err
			}

			utils.Infof("Sandbox %s connected successfully", sandboxID)

			// 立即請求狀態更新
			statusRequest := &pb.SchedulerMessage{
				SandboxId: sandboxID,
				MessageType: &pb.SchedulerMessage_StatusRequest{
					StatusRequest: &pb.SandboxStatusRequest{},
				},
			}

			if err := stream.Send(statusRequest); err != nil {
				utils.Errorf("Failed to send initial status request: %v", err)
			} else {
				utils.Debugf("Requested initial status from sandbox %s", sandboxID)
			}

			// 啟動任務發送 goroutine
			go s.sendJobsToSandbox(instance)

		case *pb.SandboxMessage_Status:
			// 處理狀態更新
			if instance != nil {
				instance.Status = msgType.Status
				instance.LastSeen = time.Now()
				utils.Debugf("Received status from sandbox %s - Available: %d, Waiting: %d, Processing: %d, Total: %d",
					sandboxID, msgType.Status.AvailableCount, msgType.Status.WaitingCount,
					msgType.Status.ProcessingCount, msgType.Status.TotalCount)
			}

		case *pb.SandboxMessage_JobResponse:
			// 處理任務響應
			jobResp := msgType.JobResponse
			utils.Infof("Job response from sandbox %s: Success=%t, Message=%s",
				sandboxID, jobResp.Success, jobResp.Message)
		}
	}

	return nil
}

// sendJobsToSandbox 發送任務到沙箱
func (s *SandboxScheduler) sendJobsToSandbox(instance *SandboxInstance) {
	for jobReq := range instance.JobChan {
		message := &pb.SchedulerMessage{
			SandboxId: instance.ID,
			MessageType: &pb.SchedulerMessage_JobRequest{
				JobRequest: jobReq,
			},
		}

		if err := instance.Stream.Send(message); err != nil {
			utils.Errorf("Failed to send job to sandbox %s: %v", instance.ID, err)
			// 將任務放回隊列或標記為失敗
			break
		}

		utils.Debugf("Sent job to sandbox %s", instance.ID)
	}
}

// GetBestSandbox 根據負載選擇最佳的沙箱實例
func (s *SandboxScheduler) GetBestSandbox() *SandboxInstance {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var candidates []*SandboxInstance
	for _, instance := range s.instances {
		if instance.Active && instance.Status != nil {
			candidates = append(candidates, instance)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// 按可用容量排序，選擇可用容量最多的
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Status.AvailableCount > candidates[j].Status.AvailableCount
	})

	return candidates[0]
}

// ReserveJob 添加任務到最佳沙箱
func (s *SandboxScheduler) ReserveJob(parentGitFullName string, gitRepoURL string, gitFullName string, gitAfterHash string, gitUsername string, gitToken string, userQuestionTableID uint64) error {
	instance := s.GetBestSandbox()
	if instance == nil {
		return fmt.Errorf("no available sandbox instances")
	}

	jobReq := &pb.AddJobRequest{
		ParentGitFullName:   parentGitFullName,
		GitRepoUrl:          gitRepoURL,
		GitFullName:         gitFullName,
		GitAfterHash:        gitAfterHash,
		GitUsername:         gitUsername,
		GitToken:            gitToken,
		UserQuestionTableId: userQuestionTableID,
	}

	// 非阻塞發送到任務通道
	select {
	case instance.JobChan <- jobReq:
		utils.Debugf("Job queued for sandbox %s", instance.ID)
		return nil
	default:
		return fmt.Errorf("sandbox %s job queue is full", instance.ID)
	}
}

// GetGlobalStatus 獲取所有沙箱的全局狀態
func (s *SandboxScheduler) GetGlobalStatus() *pb.SandboxStatusResponse {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var totalAvailable, totalWaiting, totalProcessing, totalCount int32

	for _, instance := range s.instances {
		if instance.Active && instance.Status != nil {
			totalAvailable += instance.Status.AvailableCount
			totalWaiting += instance.Status.WaitingCount
			totalProcessing += instance.Status.ProcessingCount
			totalCount += instance.Status.TotalCount
		}
	}

	return &pb.SandboxStatusResponse{
		AvailableCount:  totalAvailable,
		WaitingCount:    totalWaiting,
		ProcessingCount: totalProcessing,
		TotalCount:      totalCount,
	}
}

// GetActiveInstanceCount 獲取活躍實例數量
func (s *SandboxScheduler) GetActiveInstanceCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	count := 0
	for _, instance := range s.instances {
		if instance.Active {
			count++
		}
	}
	return count
}

// cleanupInactiveInstances 清理不活躍的實例
func (s *SandboxScheduler) cleanupInactiveInstances() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mutex.Lock()
		now := time.Now()
		for id, instance := range s.instances {
			// 如果超過 1 分鐘沒有狀態更新，標記為不活躍
			if now.Sub(instance.LastSeen) > time.Minute {
				if instance.Active {
					utils.Warnf("Marking sandbox %s as inactive due to missing status update", id)
					instance.Active = false
				}

				// 如果超過 5 分鐘沒有狀態更新，完全移除
				if now.Sub(instance.LastSeen) > 5*time.Minute {
					utils.Infof("Removing inactive sandbox %s", id)
					close(instance.JobChan)
					delete(s.instances, id)
				}
			}
		}
		s.mutex.Unlock()
	}
}

// Close 關閉調度器
func (s *SandboxScheduler) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, instance := range s.instances {
		close(instance.JobChan)
	}
	s.instances = make(map[string]*SandboxInstance)
}
