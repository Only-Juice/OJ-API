# OJ-API Sandbox gRPC 分離架構

## 架構概述

此項目已將sandbox功能透過gRPC分離為獨立的服務，提供更好的可擴展性和維護性。

### 架構組件

1. **主API服務器** (`main.go`) - 處理HTTP API請求
2. **Sandbox gRPC服務器** (`cmd/sandbox-server/main.go`) - 處理代碼執行任務
3. **gRPC客戶端管理器** (`services/sandbox_client_manager.go`) - 管理與sandbox服務的通信

## 部署方式

### 1. 使用Docker Compose (推薦)

```bash
# 構建並啟動所有服務
docker-compose up --build

# 後台運行
docker-compose up -d --build
```

### 2. 手動部署

#### 啟動Sandbox gRPC服務器

```bash
# 編譯sandbox服務器
go build -o sandbox-server ./cmd/sandbox-server

# 啟動sandbox服務器 (需要isolate環境)
./sandbox-server
```

#### 啟動主API服務器

```bash
# 設置環境變數
export SANDBOX_GRPC_ADDRESS=localhost:50051

# 編譯主程序
go build -o main .

# 啟動主服務器
./main
```

## 環境變數配置

### 主API服務器

```env
# Sandbox gRPC 服務器地址
SANDBOX_GRPC_ADDRESS=localhost:50051

# 其他原有配置...
API_PORT=8080
DB_HOST=localhost
DB_PORT=5432
# ...
```

### Sandbox服務器

```env
# Sandbox實例數量
SANDBOX_COUNT=4

# 數據庫配置（如果需要）
DB_HOST=localhost
DB_PORT=5432
# ...
```

## gRPC服務接口

### SandboxService

```protobuf
service SandboxService {
  // 執行代碼
  rpc ExecuteCode(ExecuteCodeRequest) returns (ExecuteCodeResponse);
  
  // 獲取沙箱狀態
  rpc GetStatus(SandboxStatusRequest) returns (SandboxStatusResponse);
  
  // 添加任務到隊列
  rpc AddJob(AddJobRequest) returns (AddJobResponse);
  
  // 健康檢查
  rpc HealthCheck(SandboxStatusRequest) returns (SandboxStatusResponse);
}
```

## 代碼變更說明

### 1. 主要變更

- 移除了對 `sandbox.SandboxPtr` 的直接依賴
- 添加了 `services.SandboxClientManager` 全局客戶端管理器
- 所有sandbox操作現在通過gRPC調用

### 2. 受影響的處理器

- `handlers/sandbox.go` - 狀態查詢使用gRPC
- `handlers/score.go` - 任務提交使用gRPC  
- `handlers/webhook.go` - 任務提交使用gRPC

### 3. 新增文件

- `proto/sandbox.proto` - gRPC服務定義
- `proto/sandbox.pb.go` - 生成的protobuf代碼
- `proto/sandbox_grpc.pb.go` - 生成的gRPC代碼
- `services/sandbox_service.go` - gRPC服務實現
- `services/sandbox_client_manager.go` - 客戶端管理器
- `cmd/sandbox-server/main.go` - 獨立sandbox服務器

## 優勢

1. **服務分離**: sandbox邏輯獨立運行，可單獨擴展
2. **故障隔離**: sandbox服務崩潰不會影響主API
3. **水平擴展**: 可以運行多個sandbox服務實例
4. **資源管理**: sandbox服務可部署在專用的高性能機器上
5. **維護性**: 代碼結構更清晰，職責分離

## 監控和調試

### 檢查服務狀態

```bash
# 檢查sandbox服務健康狀態
curl http://localhost:8080/api/sandbox/status
```

### 日誌查看

```bash
# Docker環境
docker-compose logs sandbox-server
docker-compose logs api-server

# 本地運行
# 查看各服務的輸出日誌
```

## 故障排除

### 常見問題

1. **連接失敗**: 檢查 `SANDBOX_GRPC_ADDRESS` 配置
2. **isolate權限**: sandbox服務需要適當的系統權限
3. **端口衝突**: 確保50051端口未被佔用

### 檢查連接

```bash
# 測試gRPC連接
grpcurl -plaintext localhost:50051 list
```
