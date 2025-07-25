package services

import (
	"OJ-API/models"
	"OJ-API/sandbox"
	pb "OJ-API/proto"
	"context"
	"fmt"
	"log"
	"time"

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

// ExecuteCode 執行代碼
func (s *SandboxServer) ExecuteCode(ctx context.Context, req *pb.ExecuteCodeRequest) (*pb.ExecuteCodeResponse, error) {
	log.Printf("Received ExecuteCode request for repo: %s", req.Repo)

	// 創建 UserQuestionTable 模型
	uqr := models.UserQuestionTable{
		ID: uint(req.UserQuestionTableId),
	}

	// 將任務添加到隊列
	s.sandbox.ReserveJob(req.Repo, req.CodePath, uqr)

	return &pb.ExecuteCodeResponse{
		Success: true,
		Message: "Code execution job queued successfully",
		Score:   0, // 初始分數，會在實際執行後更新
		JudgeTime: time.Now().Format(time.RFC3339),
	}, nil
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
	// 驗證請求
	if req.Repo == "" {
		return nil, status.Errorf(codes.InvalidArgument, "repo cannot be empty")
	}

	// 創建 UserQuestionTable 模型
	uqr := models.UserQuestionTable{
		ID: uint(req.UserQuestionTableId),
	}

	// 添加任務到隊列
	s.sandbox.ReserveJob(req.Repo, req.CodePath, uqr)

	return &pb.AddJobResponse{
		Success: true,
		Message: "Job added to queue successfully",
		JobId:   fmt.Sprintf("job_%d_%s", req.UserQuestionTableId, req.Repo),
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

// ExecuteCode 執行代碼
func (c *SandboxClient) ExecuteCode(ctx context.Context, repo string, codePath []byte, userQuestionTableID uint64) (*pb.ExecuteCodeResponse, error) {
	req := &pb.ExecuteCodeRequest{
		Repo:                  repo,
		CodePath:              codePath,
		UserQuestionTableId:   userQuestionTableID,
	}
	
	return c.client.ExecuteCode(ctx, req)
}

// GetStatus 獲取沙箱狀態
func (c *SandboxClient) GetStatus(ctx context.Context) (*pb.SandboxStatusResponse, error) {
	req := &pb.SandboxStatusRequest{}
	return c.client.GetStatus(ctx, req)
}

// AddJob 添加任務
func (c *SandboxClient) AddJob(ctx context.Context, repo string, codePath []byte, userQuestionTableID uint64) (*pb.AddJobResponse, error) {
	req := &pb.AddJobRequest{
		Repo:                  repo,
		CodePath:              codePath,
		UserQuestionTableId:   userQuestionTableID,
	}
	
	return c.client.AddJob(ctx, req)
}

// HealthCheck 健康檢查
func (c *SandboxClient) HealthCheck(ctx context.Context) (*pb.SandboxStatusResponse, error) {
	req := &pb.SandboxStatusRequest{}
	return c.client.HealthCheck(ctx, req)
}
