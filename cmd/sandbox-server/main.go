package main

import (
	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	pb "OJ-API/proto"
	"OJ-API/sandbox"
	"OJ-API/utils"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	// 初始化日誌
	utils.InitLog()

	// 加載環境變數
	if err := godotenv.Load(".env.local"); err != nil {
		utils.Info("No .env.local file found")
	}

	// 初始化數據庫連接
	if err := database.Connect(); err != nil {
		utils.Fatalf("Failed to connect to database: %v", err)
	}

	// 創建沙箱實例
	sandboxCount := runtime.NumCPU()
	if countStr := config.Config("SANDBOX_COUNT"); countStr != "" {
		if count, err := strconv.Atoi(countStr); err == nil {
			sandboxCount = count
		}
	}

	sandboxInstance := sandbox.NewSandbox(sandboxCount)
	defer sandboxInstance.Cleanup()

	// 啟動工作循環
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sandboxInstance.WorkerLoop(ctx)

	// 生成唯一的沙箱 ID
	sandboxID := uuid.New().String()

	// 連接到 API Server 調度器
	schedulerAddress := config.Config("OJ_HOST")
	if schedulerAddress == "" {
		schedulerAddress = "localhost:3001"
	}

	conn, err := grpc.Dial(schedulerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		utils.Fatalf("Failed to connect to scheduler: %v", err)
	}
	defer conn.Close()

	schedulerClient := pb.NewSchedulerServiceClient(conn)

	// 建立雙向流連接
	stream, err := schedulerClient.SandboxStream(context.Background())
	if err != nil {
		utils.Fatalf("Failed to create stream: %v", err)
	}

	utils.Debugf("Sandbox %s connecting to scheduler at %s...", sandboxID, schedulerAddress)

	// 發送連接請求
	connectMsg := &pb.SandboxMessage{
		SandboxId: sandboxID,
		MessageType: &pb.SandboxMessage_Connect{
			Connect: &pb.SandboxConnectRequest{
				SandboxId: sandboxID,
				Capacity:  int32(sandboxCount),
			},
		},
	}

	if err := stream.Send(connectMsg); err != nil {
		utils.Fatalf("Failed to send connect message: %v", err)
	}

	// 啟動消息處理 goroutine
	go handleSchedulerMessages(stream, sandboxInstance)

	// 啟動狀態更新 goroutine
	go sendStatusUpdates(stream, sandboxID, sandboxInstance)

	// 優雅關機處理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	utils.Info("Shutting down sandbox server...")
	cancel() // 停止工作循環
	stream.CloseSend()
}

// handleSchedulerMessages 處理來自調度器的消息
func handleSchedulerMessages(stream pb.SchedulerService_SandboxStreamClient, sandboxInstance *sandbox.Sandbox) {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			utils.Debugf("Stream closed by scheduler")
			break
		}
		if err != nil {
			utils.Debugf("Stream receive error: %v", err)
			break
		}

		switch msgType := msg.MessageType.(type) {
		case *pb.SchedulerMessage_ConnectResponse:
			resp := msgType.ConnectResponse
			if resp.Success {
				utils.Debugf("Successfully connected to scheduler: %s", resp.Message)
				// 連接成功後立即發送初始狀態
				sendCurrentStatus(stream, msg.SandboxId, sandboxInstance)
			} else {
				utils.Errorf("Failed to connect to scheduler: %s", resp.Message)
			}

		case *pb.SchedulerMessage_JobRequest:
			// 處理任務請求
			jobReq := msgType.JobRequest
			utils.Debugf("Received job request for repo: %s, commit: %s", jobReq.GitFullName, jobReq.GitAfterHash)

			// 異步處理任務
			go func() {
				// 發送任務響應
				responseMsg, err := AddJob(sandboxInstance, context.Background(), jobReq)
				if err != nil {
					utils.Errorf("Failed to add job: %v", err)
					return
				}

				// 將 AddJobResponse 包裝在 SandboxMessage 中
				sandboxMsg := &pb.SandboxMessage{
					SandboxId: msg.SandboxId,
					MessageType: &pb.SandboxMessage_JobResponse{
						JobResponse: responseMsg,
					},
				}

				if err := stream.Send(sandboxMsg); err != nil {
					utils.Debugf("Failed to send job response: %v", err)
				} else {
					utils.Debugf("Successfully sent job response")
				}
			}()

		case *pb.SchedulerMessage_StatusRequest:
			// 處理狀態請求 - 立即發送狀態
			sendCurrentStatus(stream, msg.SandboxId, sandboxInstance)
		}
	}
}

// sendStatusUpdates 定期發送狀態更新
func sendStatusUpdates(stream pb.SchedulerService_SandboxStreamClient, sandboxID string, sandboxInstance *sandbox.Sandbox) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		sendCurrentStatus(stream, sandboxID, sandboxInstance)
	}
}

var lastStatus = struct {
	lastAvailable  int32
	lastWaiting    int32
	lastProcessing int32
	lastTotal      int32
	lastSendTime   time.Time
}{}

// sendCurrentStatus 發送當前狀態
func sendCurrentStatus(stream pb.SchedulerService_SandboxStreamClient, sandboxID string, sandboxInstance *sandbox.Sandbox) {
	available := int32(sandboxInstance.AvailableCount())
	waiting := int32(sandboxInstance.WaitingCount())
	processing := int32(sandboxInstance.ProcessingCount())
	total := available + processing

	now := time.Now()

	// 檢查狀態是否有變化或超過15秒沒有發送
	statusChanged := available != lastStatus.lastAvailable ||
		waiting != lastStatus.lastWaiting ||
		processing != lastStatus.lastProcessing ||
		total != lastStatus.lastTotal

	timeSinceLastSend := now.Sub(lastStatus.lastSendTime)
	forceUpdate := timeSinceLastSend >= 15*time.Second

	if !statusChanged && !forceUpdate {
		return // 沒有變化且未超過15秒，不發送
	}

	// 更新上次狀態和發送時間
	lastStatus.lastAvailable = available
	lastStatus.lastWaiting = waiting
	lastStatus.lastProcessing = processing
	lastStatus.lastTotal = total
	lastStatus.lastSendTime = now

	status := &pb.SandboxStatusResponse{
		AvailableCount:  available,
		WaitingCount:    waiting,
		ProcessingCount: processing,
		TotalCount:      total,
	}

	statusMsg := &pb.SandboxMessage{
		SandboxId: sandboxID,
		MessageType: &pb.SandboxMessage_Status{
			Status: status,
		},
	}

	if err := stream.Send(statusMsg); err != nil {
		utils.Debugf("Failed to send status update: %v", err)
	} else {
		if forceUpdate {
			utils.Debugf("Force sent status update after 15s - Available: %d, Waiting: %d, Processing: %d, Total: %d",
				available, waiting, processing, total)
		} else {
			utils.Debugf("Sent status update - Available: %d, Waiting: %d, Processing: %d, Total: %d",
				available, waiting, processing, total)
		}
	}
}

// AddJob 添加任務到隊列
func AddJob(sandboxInstance *sandbox.Sandbox, ctx context.Context, req *pb.AddJobRequest) (*pb.AddJobResponse, error) {
	// 創建 UserQuestionTable 模型
	uqr := models.UserQuestionTable{
		ID: uint(req.UserQuestionTableId),
	}

	codePath, err := CloneRepository(req.GitFullName, req.GitRepoUrl, req.GitAfterHash, req.GitUsername, req.GitToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clone repository: %v", err)
	}

	// 添加任務到隊列
	sandboxInstance.ReserveJob(req.ParentGitFullName, []byte(codePath), uqr)

	return &pb.AddJobResponse{
		Success: true,
		Message: "Job added to queue successfully",
		JobId:   fmt.Sprintf("job_%d_%s", req.UserQuestionTableId, req.GitFullName),
	}, nil
}

// CloneRepository 執行 git clone 操作
func CloneRepository(GitFullName, GitRepoURL, GitAfterHash, GitUsername, GitToken string) (string, error) {
	// 生成唯一的代碼路徑
	codePath := fmt.Sprintf("%s/%s", config.Config("REPO_FOLDER"), GitFullName+"/"+uuid.New().String())

	// 配置 clone 選項
	cloneOptions := &git.CloneOptions{
		URL: GitRepoURL,
		Auth: &http.BasicAuth{
			Username: GitUsername,
			Password: GitToken,
		},
		Progress: nil, // 在生產環境中不輸出進度
	}

	// 執行 clone
	repo, err := git.PlainClone(codePath, false, cloneOptions)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %v", err)
	}

	// 如果有指定的 commit hash，則 checkout 到該 commit
	if GitAfterHash != "" {
		// 獲取 worktree 並 checkout 到指定 commit
		worktree, err := repo.Worktree()
		if err != nil {
			return "", fmt.Errorf("failed to get worktree: %v", err)
		}

		// Checkout 到指定的 commit hash
		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(GitAfterHash),
		})
		if err != nil {
			return "", fmt.Errorf("failed to checkout to %s: %v", GitAfterHash, err)
		}
		utils.Debugf("Successfully cloned and checked out %s to %s at commit %s", GitFullName, codePath, GitAfterHash)
	} else {
		utils.Debugf("Successfully cloned %s to %s (using HEAD)", GitFullName, codePath)
	}

	// 設置目錄權限為 777 (讀寫執行權限)
	err = os.Chmod(codePath, 0777)
	if err != nil {
		utils.Warnf("Warning: failed to set permissions for %s: %v", codePath, err)
	}

	// 遞歸設置所有子目錄和文件的權限
	err = filepath.Walk(codePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Chmod(path, 0777)
		}
		return os.Chmod(path, 0644)
	})
	if err != nil {
		utils.Warnf("Warning: failed to set recursive permissions for %s: %v", codePath, err)
	}

	return codePath, nil
}
