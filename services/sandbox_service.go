package services

import (
	"OJ-API/config"
	"OJ-API/models"
	pb "OJ-API/proto"
	"OJ-API/sandbox"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// SandboxServer 實現 gRPC sandbox 服務
type SandboxServer struct {
	pb.UnimplementedSandboxServiceServer
	sandbox *sandbox.Sandbox
}

// NewSandboxServer 創建新的 sandbox gRPC 服務器
func NewSandboxServer(sandboxInstance *sandbox.Sandbox) *SandboxServer {
	return &SandboxServer{
		sandbox: sandboxInstance,
	}
}

// GetStatus 獲取沙箱狀態
func (s *SandboxServer) GetStatus(ctx context.Context, req *pb.SandboxStatusRequest) (*pb.SandboxStatusResponse, error) {
	return &pb.SandboxStatusResponse{
		AvailableCount:  int32(s.sandbox.AvailableCount()),
		WaitingCount:    int32(s.sandbox.WaitingCount()),
		ProcessingCount: int32(s.sandbox.ProcessingCount()),
		TotalCount:      int32(s.sandbox.AvailableCount() + s.sandbox.ProcessingCount()),
	}, nil
}

// AddJob 添加任務到隊列
func (s *SandboxServer) AddJob(ctx context.Context, req *pb.AddJobRequest) (*pb.AddJobResponse, error) {
	// 創建 UserQuestionTable 模型
	uqr := models.UserQuestionTable{
		ID: uint(req.UserQuestionTableId),
	}

	codePath, err := CloneRepository(req.GitFullName, req.GitRepoUrl, req.GitAfterHash, req.GitUsername, req.GitToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clone repository: %v", err)
	}

	// 添加任務到隊列
	s.sandbox.ReserveJob(req.ParentGitFullName, []byte(codePath), uqr)

	return &pb.AddJobResponse{
		Success: true,
		Message: "Job added to queue successfully",
		JobId:   fmt.Sprintf("job_%d_%s", req.UserQuestionTableId, req.GitFullName),
	}, nil
}

// HealthCheck 健康檢查
func (s *SandboxServer) HealthCheck(ctx context.Context, req *pb.SandboxStatusRequest) (*pb.SandboxStatusResponse, error) {
	return s.GetStatus(ctx, req)
}

// SandboxClient gRPC 客戶端包裝器
type SandboxClient struct {
	client pb.SandboxServiceClient
	conn   *grpc.ClientConn
}

// NewSandboxClient 創建新的 sandbox gRPC 客戶端
func NewSandboxClient(address string) (*SandboxClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sandbox service: %v", err)
	}

	client := pb.NewSandboxServiceClient(conn)
	return &SandboxClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close 關閉客戶端連接
func (c *SandboxClient) Close() error {
	return c.conn.Close()
}

// GetStatus 獲取沙箱狀態
func (c *SandboxClient) GetStatus(ctx context.Context) (*pb.SandboxStatusResponse, error) {
	req := &pb.SandboxStatusRequest{}
	return c.client.GetStatus(ctx, req)
}

// AddJob 添加任務
func (c *SandboxClient) AddJob(ctx context.Context, parentGitFullName string, gitRepoURL string, gitFullName string, gitAfterHash string, gitUsername string, gitToken string, userQuestionTableID uint64) (*pb.AddJobResponse, error) {
	req := &pb.AddJobRequest{
		ParentGitFullName:   parentGitFullName,
		GitRepoUrl:          gitRepoURL,
		GitFullName:         gitFullName,
		GitAfterHash:        gitAfterHash,
		GitUsername:         gitUsername,
		GitToken:            gitToken,
		UserQuestionTableId: userQuestionTableID,
	}

	return c.client.AddJob(ctx, req)
}

// HealthCheck 健康檢查
func (c *SandboxClient) HealthCheck(ctx context.Context) (*pb.SandboxStatusResponse, error) {
	req := &pb.SandboxStatusRequest{}
	return c.client.HealthCheck(ctx, req)
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
		log.Printf("Successfully cloned and checked out %s to %s at commit %s", GitFullName, codePath, GitAfterHash)
	} else {
		log.Printf("Successfully cloned %s to %s (using HEAD)", GitFullName, codePath)
	}

	// 設置目錄權限為 777 (讀寫執行權限)
	err = os.Chmod(codePath, 0777)
	if err != nil {
		log.Printf("Warning: failed to set permissions for %s: %v", codePath, err)
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
		log.Printf("Warning: failed to set recursive permissions for %s: %v", codePath, err)
	}

	return codePath, nil
}
