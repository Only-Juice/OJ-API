# 沙箱 gRPC 架構變更 - 統一端口雙向流版本

## 概述

本次變更採用雙向流（bidirectional streaming）架構，並將 HTTP 和 gRPC 服務統一到同一端口，進一步簡化了端口管理：

1. **API Server** 在單一端口同時運行 HTTP API 和 gRPC 調度器服務 (SchedulerService)
2. **沙箱服務器** 作為客戶端主動連接到 API Server - **不再開放任何端口**
3. 通過單一雙向流連接處理所有通信（註冊、心跳、任務分發）
4. HTTP 和 gRPC 請求通過協議檢測自動路由到對應處理器

## 架構圖

```
[API Server - 統一端口 (API_PORT)]
├── HTTP API 服務 (Gin)
└── gRPC 調度器服務 (SchedulerService)
                    ↑
                    │ 雙向流連接 (統一端口)
                    │
    ┌───────────────┼───────────────┐
    │               │               │ 
[Sandbox Client] [Sandbox Client] [Sandbox Client]
  (無端口)         (無端口)         (無端口)
```

## 主要優勢

### 🎯 **端口極度簡化**
- **之前**: API Server HTTP (8080) + gRPC Scheduler (50052) + 每個沙箱 (50051, 50053, 50054...)
- **現在**: 僅需單一端口 (API_PORT) 同時處理 HTTP 和 gRPC

### 🔀 **智能協議路由**
- 自動檢測 HTTP/gRPC 請求類型
- 無需額外配置或代理
- 支持 HTTP/2 和 gRPC 共存

### 🔄 **雙向流通信**
- 單一連接處理所有消息類型
- 實時任務分發和狀態更新
- 無需額外的心跳機制

### 🚀 **部署簡化**
- 防火牆配置更簡單
- 負載均衡器配置統一
- 沙箱自動連接和註冊

## 主要變更

### 1. 新增雙向流服務

在 `proto/sandbox.proto` 中新增：

```protobuf
service SchedulerService {
  // 沙箱雙向流連接
  rpc SandboxStream(stream SandboxMessage) returns (stream SchedulerMessage);
}
```

### 2. 消息類型

- **SandboxMessage**: 沙箱→調度器 (連接請求、狀態更新、任務響應)
- **SchedulerMessage**: 調度器→沙箱 (連接響應、任務請求、狀態查詢)

### 3. 沙箱服務器變更

- 移除 gRPC 服務器代碼
- 改為純客戶端模式
- 建立雙向流連接到調度器

- 管理多個沙箱實例
- 根據負載選擇最佳沙箱
- 自動清理不活躍的實例
- 提供全局狀態統計

### 3. 修改 API Server

- 在單一端口同時支持 HTTP 和 gRPC 服務
- 使用 HTTP/2 協議檢測自動路由請求
- 集成沙箱調度管理
- 優雅關機處理

### 4. 修改沙箱服務器

- 主動連接到 API Server 調度器
- 自動註冊和心跳維持
- 支援配置化端口和地址

## 配置參數

### API Server (.env.local)

```bash
# 統一端口 (同時處理 HTTP API 和 gRPC)
API_PORT=8080

# 注意：不再需要單獨的 GRPC_PORT 配置
```

### 沙箱服務器

```bash
# 沙箱容量
SANDBOX_COUNT=4

# 調度器地址 (使用 API Server 的統一端口)
SCHEDULER_ADDRESS=localhost:8080

# 注意：不再需要 SANDBOX_PORT 和 SANDBOX_EXTERNAL_ADDRESS
# 沙箱服務器不再開放任何端口
# SANDBOX_ID 會自動使用 UUID 生成，無需手動配置
```

## 使用方式

### 1. 啟動 API Server

```bash
go run main.go
```

這將啟動：
- HTTP API 服務器和 gRPC 調度器服務 (統一端口 8080)
- 自動協議檢測和路由

### 2. 啟動沙箱服務器

#### 單個沙箱

```bash
cd cmd/sandbox-server
go run main.go
```

#### 多個沙箱 (使用腳本)

```bash
./start_sandboxes.sh
```

這將啟動 3 個沙箱服務器實例：
- 沙箱 1: 自動生成 UUID 作為實例 ID
- 沙箱 2: 自動生成 UUID 作為實例 ID  
- 沙箱 3: 自動生成 UUID 作為實例 ID

所有沙箱都會自動連接到 API Server 的統一端口 (8080)。

#### 停止所有沙箱

```bash
./stop_sandboxes.sh
```

### 3. 手動啟動多個沙箱

```bash
# 終端 1
go run cmd/sandbox-server/main.go

# 終端 2  
go run cmd/sandbox-server/main.go

# 終端 3
go run cmd/sandbox-server/main.go
```

每個實例啟動時會自動生成唯一的 UUID 作為實例 ID，所有實例都會自動連接到 `localhost:8080` (API Server 統一端口)。

## 功能特性

### 1. 自動負載平衡

調度器會根據各沙箱的可用容量自動選擇最佳實例。

### 2. 健康檢查

- 沙箱每 15 秒發送心跳
- 超過 1 分鐘無心跳標記為不活躍  
- 超過 5 分鐘移除實例

### 3. 動態擴縮容

- 可隨時新增沙箱實例
- 實例自動註冊到調度器
- 支援實例動態下線

### 4. 錯誤處理

- 連接失敗自動重試
- 實例故障自動剔除
- 優雅關機處理

## API 變更

原有的 API 接口保持不變，底層會自動使用新的調度機制：

- `POST /sandbox/reserve` - 任務提交 (自動調度到最佳沙箱)
- `GET /sandbox/status` - 獲取全局狀態

## 監控和除錯

### 查看調度器狀態

API Server 日誌會顯示：
- 沙箱註冊/註銷事件
- 心跳接收情況
- 任務調度決策

### 查看沙箱狀態

沙箱服務器日誌會顯示：
- 註冊結果
- 心跳發送狀態
- 任務執行情況

## 遷移指南

### 從舊架構遷移

1. 停止舊的沙箱服務器
2. 更新 `.env.local` 配置
3. 啟動新的 API Server 
4. 啟動新的沙箱服務器

### 回滾方案

如需回滾到舊架構：
1. 停止所有服務
2. 恢復舊版本的代碼
3. 使用舊的配置檔案啟動

## 故障排除

### 常見問題

1. **沙箱無法註冊**
   - 檢查 `SCHEDULER_ADDRESS` 配置 (應為 API Server 端口，如 localhost:8080)
   - 確認 API Server 已啟動並且 HTTP/gRPC 服務正常運行

2. **HTTP 請求被路由到 gRPC**
   - 檢查請求 Content-Type 和協議版本
   - 確認客戶端使用正確的 HTTP/1.1 或 HTTP/2 協議

3. **gRPC 請求失敗**
   - 確認 gRPC 客戶端使用 HTTP/2 協議
   - 檢查 Content-Type 是否包含 "application/grpc"

4. **負載不均衡**  
   - 檢查沙箱心跳是否正常
   - 查看調度器日誌中的選擇邏輯

5. **連接超時**
   - 增加超時時間配置
   - 檢查網路連通性
   - 確認防火牆僅開放 API_PORT

### 日誌位置

- API Server: 標準輸出
- 沙箱服務器: 標準輸出
- 建議使用 systemd 或 Docker 進行日誌管理
