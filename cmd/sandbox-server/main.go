package main

import (
	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	pb "OJ-API/proto"
	"OJ-API/sandbox"
	"OJ-API/utils"
	"context"
	"crypto/tls"
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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	// 初始化日誌
	utils.InitLog()
	utils.SetServerSource("sandbox-server")

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
	schedulerAddress := config.Config("SCHEDULER_ADDRESS")
	if schedulerAddress == "" {
		schedulerAddress = "localhost:3001"
	}

	// 優雅關機處理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 啟動重連循環
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := connectToScheduler(schedulerAddress, sandboxID)
				if err != nil {
					utils.Errorf("Failed to connect to scheduler: %v", err)
					select {
					case <-ctx.Done():
						return
					case <-time.After(5 * time.Second):
						continue
					}
				}

				// 連接成功，處理連接直到斷線
				err = handleConnection(ctx, conn, sandboxID, sandboxInstance)
				if err != nil {
					utils.Errorf("Connection lost: %v", err)
				}
				conn.Close()

				// 連接斷開，等待重連
				select {
				case <-ctx.Done():
					return
				case <-time.After(3 * time.Second):
					utils.Info("Attempting to reconnect to scheduler...")
				}
			}
		}
	}()

	<-sigChan
	utils.Info("Shutting down sandbox server...")
	cancel() // 停止工作循環

	// 等待所有任務完成，但設置超時限制
	utils.Info("Waiting for all jobs to complete...")
	shutdownTimeoutStr := config.Config("SHUTDOWN_TIMEOUT")
	shutdownTimeout := 30 * time.Second
	if shutdownTimeoutStr != "" {
		if timeout, err := time.ParseDuration(shutdownTimeoutStr); err == nil {
			shutdownTimeout = timeout
		}
	}

	shutdownStart := time.Now()
	lastReportTime := time.Now()

	for {
		waitingCount := sandboxInstance.WaitingCount()
		processingCount := sandboxInstance.ProcessingCount()

		if waitingCount == 0 && processingCount == 0 {
			utils.Info("All jobs completed, shutting down...")
			break
		}

		// 每10秒報告一次進度
		if time.Since(lastReportTime) > 10*time.Second {
			utils.Infof("Still waiting for jobs to complete - Waiting: %d, Processing: %d",
				waitingCount, processingCount)
			lastReportTime = time.Now()
		}

		// 檢查是否超時
		if time.Since(shutdownStart) > shutdownTimeout {
			utils.Warnf("Shutdown timeout reached (%v), forcing shutdown with %d jobs still processing...",
				shutdownTimeout, processingCount)
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// connectToScheduler 連接到調度器並建立流
func connectToScheduler(schedulerAddress, sandboxID string) (*grpc.ClientConn, error) {
	utils.Debugf("Sandbox %s connecting to scheduler at %s...", sandboxID, schedulerAddress)

	// 檢查是否使用 HTTPS
	useTLS := config.Config("USE_TLS")
	var creds credentials.TransportCredentials

	if useTLS == "true" {
		// 載入 TLS 憑證
		certFile := config.Config("TLS_CERT_FILE")
		keyFile := config.Config("TLS_KEY_FILE")

		if certFile == "" {
			certFile = "cert.pem"
		}
		if keyFile == "" {
			keyFile = "key.pem"
		}

		// 載入客戶端憑證
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificates: %v", err)
		}

		// 配置 TLS
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   config.Config("TLS_SERVER_NAME"), // 如果需要指定服務器名稱
		}

		// 如果設定跳過憑證驗證（僅用於開發環境）
		if config.Config("TLS_SKIP_VERIFY") == "true" {
			tlsConfig.InsecureSkipVerify = true
			utils.Warnf("TLS certificate verification is disabled - only use in development!")
		}

		creds = credentials.NewTLS(tlsConfig)
		utils.Debugf("Using HTTPS connection with TLS certificates")
	} else {
		creds = insecure.NewCredentials()
		utils.Debugf("Using HTTP connection without TLS")
	}

	conn, err := grpc.NewClient(schedulerAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to dial scheduler: %v", err)
	}

	return conn, nil
}

// handleConnection 處理與調度器的連接
func handleConnection(ctx context.Context, conn *grpc.ClientConn, sandboxID string, sandboxInstance *sandbox.Sandbox) error {
	schedulerClient := pb.NewSchedulerServiceClient(conn)

	// 建立雙向流連接
	stream, err := schedulerClient.SandboxStream(ctx)
	if err != nil {
		return fmt.Errorf("failed to create stream: %v", err)
	}

	// 發送連接請求
	connectMsg := &pb.SandboxMessage{
		SandboxId: sandboxID,
		MessageType: &pb.SandboxMessage_Connect{
			Connect: &pb.SandboxConnectRequest{
				SandboxId: sandboxID,
				Capacity:  int32(sandboxInstance.AvailableCount() + sandboxInstance.ProcessingCount()),
			},
		},
	}

	if err := stream.Send(connectMsg); err != nil {
		return fmt.Errorf("failed to send connect message: %v", err)
	}

	// 創建用於停止 goroutines 的 context
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	// 啟動消息處理 goroutine
	messageDone := make(chan error, 1)
	go func() {
		err := handleSchedulerMessages(stream, sandboxInstance)
		messageDone <- err
	}()

	// 啟動狀態更新 goroutine
	statusDone := make(chan error, 1)
	go func() {
		err := sendStatusUpdates(streamCtx, stream, sandboxID, sandboxInstance)
		statusDone <- err
	}()

	// 等待任一 goroutine 結束或 context 取消
	select {
	case err := <-messageDone:
		streamCancel()
		return err
	case err := <-statusDone:
		streamCancel()
		return err
	case <-ctx.Done():
		streamCancel()
		stream.CloseSend()
		return ctx.Err()
	}
}

// handleSchedulerMessages 處理來自調度器的消息
func handleSchedulerMessages(stream pb.SchedulerService_SandboxStreamClient, sandboxInstance *sandbox.Sandbox) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			utils.Debugf("Stream closed by scheduler")
			return nil
		}
		if err != nil {
			utils.Debugf("Stream receive error: %v", err)
			return err
		}

		switch msgType := msg.MessageType.(type) {
		case *pb.SchedulerMessage_ConnectResponse:
			resp := msgType.ConnectResponse
			if resp.Success {
				utils.Debugf("Successfully connected to scheduler: %s", resp.Message)
				// 連接成功後立即發送初始狀態
				if err := sendCurrentStatus(stream, msg.SandboxId, sandboxInstance); err != nil {
					utils.Debugf("Failed to send initial status: %v", err)
				}
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
			if err := sendCurrentStatus(stream, msg.SandboxId, sandboxInstance); err != nil {
				utils.Debugf("Failed to send status response: %v", err)
			}
		}
	}
}

// sendStatusUpdates 定期發送狀態更新
func sendStatusUpdates(ctx context.Context, stream pb.SchedulerService_SandboxStreamClient, sandboxID string, sandboxInstance *sandbox.Sandbox) error {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := sendCurrentStatus(stream, sandboxID, sandboxInstance); err != nil {
				return err
			}
		}
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
func sendCurrentStatus(stream pb.SchedulerService_SandboxStreamClient, sandboxID string, sandboxInstance *sandbox.Sandbox) error {
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
		return nil // 沒有變化且未超過15秒，不發送
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
		return err
	} else {
		if forceUpdate {
			utils.Debugf("Force sent status update after 15s - Available: %d, Waiting: %d, Processing: %d, Total: %d",
				available, waiting, processing, total)
		} else {
			utils.Debugf("Sent status update - Available: %d, Waiting: %d, Processing: %d, Total: %d",
				available, waiting, processing, total)
		}
	}

	return nil
}

// AddJob 添加任務到隊列
func AddJob(sandboxInstance *sandbox.Sandbox, ctx context.Context, req *pb.AddJobRequest) (*pb.AddJobResponse, error) {
	sandboxInstance.SubtractAvailableCount()
	// 創建 UserQuestionTable 模型
	uqr := models.UserQuestionTable{
		ID: uint(req.UserQuestionTableId),
	}

	codePath, err := CloneRepository(req.GitFullName, req.GitRepoUrl, req.GitAfterHash, req.GitUsername, req.GitToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clone repository: %v", err)
	}

	sandboxInstance.AddAvailableCount()
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
	if GitAfterHash != "" && GitAfterHash != "0000000000000000000000000000000000000000" {
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
