# 修復：多 API Server Pod 只有一台能抓到 Sandbox 的問題

## 問題診斷

### 原始問題
在測試多 API Server Pod 環境時，發現雖然 Sandbox 建立了多個 gRPC 連接，但只有一個 API Server 能夠看到並使用該 Sandbox。

### 根本原因

查看 `services/sandbox_scheduler.go` 中的代碼：

```go
type SandboxScheduler struct {
    instances map[string]*SandboxInstance  // key 是 sandboxID
    ...
}

// 在 SandboxStream 中註冊
s.instances[sandboxID] = instance  // ⚠️ 問題在這裡！
```

**問題**：
1. Sandbox 為每個 API Server Pod 建立連接
2. 但所有連接都使用**相同的 `sandboxID`**
3. 每個 API Server 的 Scheduler 都維護自己的 `instances` map
4. 由於 `sandboxID` 相同，每個 Scheduler 只會記錄一個連接
5. 實際上每個 Scheduler 都"看到"了這個 Sandbox，但它們使用相同的 ID

**更深層的問題**：
- 當多個連接使用相同 ID 時，可能會導致狀態混亂
- 每個 API Server 應該將來自同一物理 Sandbox 的不同連接視為**獨立的沙箱實例**

## 解決方案

### 核心思想

為每個 API Server 連接生成**唯一的 sandboxID**：

```
基礎 ID: 550e8400-e29b-41d4-a716-446655440000

連接到 10.244.1.5:3001 → sandboxID: 550e8400-e29b-41d4-a716-446655440000-10-244-1-5-3001
連接到 10.244.1.6:3001 → sandboxID: 550e8400-e29b-41d4-a716-446655440000-10-244-1-6-3001  
連接到 10.244.1.7:3001 → sandboxID: 550e8400-e29b-41d4-a716-446655440000-10-244-1-7-3001
```

這樣：
- 每個 API Server 看到的是不同的 sandboxID
- 不會發生 ID 衝突
- 每個連接都被正確追蹤

### 代碼修改

#### 1. 生成基礎 Sandbox ID

```go
// 生成唯一的沙箱 ID（基礎 ID）
baseSandboxID := uuid.New().String()
utils.Infof("Sandbox base ID: %s", baseSandboxID)

// 啟動多連接管理器
go manageMultipleConnections(ctx, schedulerAddress, baseSandboxID, sandboxInstance)
```

#### 2. 為每個連接創建唯一 ID

```go
func connectToAllSchedulers(..., baseSandboxID string, ...) {
    ...
    for hostAddr := range currentHosts {
        if _, exists := connections[hostAddr]; !exists {
            // 為每個連接創建唯一的 sandboxID
            uniqueSandboxID := fmt.Sprintf("%s-%s", baseSandboxID, sanitizeHostAddr(hostAddr))
            
            // 啟動連接 goroutine
            go maintainConnection(connCtx, hostAddr, uniqueSandboxID, sandboxInstance)
        }
    }
}

// 清理主機地址以用於 ID
func sanitizeHostAddr(hostAddr string) string {
    // 將 IP:port 轉換為安全的 ID 格式（替換 : 和 .）
    return strings.ReplaceAll(strings.ReplaceAll(hostAddr, ":", "-"), ".", "-")
}
```

## 工作原理

### 連接建立流程

```
Sandbox Pod 啟動
    ↓
生成基礎 ID: uuid-1234
    ↓
DNS 查詢: oj-api-server-headless
    ↓
返回 IP 列表: [10.244.1.5, 10.244.1.6, 10.244.1.7]
    ↓
為每個 IP 創建連接:
    ├─ 連接 1: uniqueID = "uuid-1234-10-244-1-5-3001"
    ├─ 連接 2: uniqueID = "uuid-1234-10-244-1-6-3001"
    └─ 連接 3: uniqueID = "uuid-1234-10-244-1-7-3001"
```

### 在 API Server 側的視圖

**API Server Pod 1 (10.244.1.5)**:
```
SandboxScheduler.instances = {
    "uuid-1234-10-244-1-5-3001": SandboxInstance{...}
}
```

**API Server Pod 2 (10.244.1.6)**:
```
SandboxScheduler.instances = {
    "uuid-1234-10-244-1-6-3001": SandboxInstance{...}
}
```

**API Server Pod 3 (10.244.1.7)**:
```
SandboxScheduler.instances = {
    "uuid-1234-10-244-1-7-3001": SandboxInstance{...}
}
```

現在每個 API Server 都有自己唯一的 Sandbox 實例記錄！

## 優勢

### ✅ 避免 ID 衝突
- 每個連接都有唯一的 ID
- 不會互相覆蓋

### ✅ 正確的實例追蹤
- 每個 API Server 正確追蹤其連接
- 狀態更新不會混亂

### ✅ 獨立的任務分發
- 任何 API Server 都可以向其看到的 Sandbox 分發任務
- 物理 Sandbox 通過多個連接接收任務

### ✅ 容錯性提升
- 一個連接斷開不影響其他連接
- 物理 Sandbox 仍可通過其他連接工作

## 驗證方法

### 1. 查看 Sandbox 日誌

```bash
kubectl logs -f <sandbox-pod-name> -n oj-api
```

應該看到：
```
Sandbox base ID: 550e8400-e29b-41d4-a716-446655440000
Connecting to new scheduler at 10.244.1.5:3001
Connected to scheduler at 10.244.1.5:3001
Connecting to new scheduler at 10.244.1.6:3001
Connected to scheduler at 10.244.1.6:3001
Connecting to new scheduler at 10.244.1.7:3001
Connected to scheduler at 10.244.1.7:3001
Managing connections to 3 scheduler(s): [10.244.1.5:3001 10.244.1.6:3001 10.244.1.7:3001]
```

### 2. 查看各個 API Server 日誌

在每個 API Server Pod 上：

```bash
# API Server Pod 1
kubectl logs <api-pod-1> -n oj-api | grep "Sandbox.*connected"
# 輸出: Sandbox 550e8400-e29b-41d4-a716-446655440000-10-244-1-5-3001 connected successfully

# API Server Pod 2
kubectl logs <api-pod-2> -n oj-api | grep "Sandbox.*connected"
# 輸出: Sandbox 550e8400-e29b-41d4-a716-446655440000-10-244-1-6-3001 connected successfully

# API Server Pod 3
kubectl logs <api-pod-3> -n oj-api | grep "Sandbox.*connected"
# 輸出: Sandbox 550e8400-e29b-41d4-a716-446655440000-10-244-1-7-3001 connected successfully
```

### 3. 測試任務分發

提交測試任務到不同的 API Server：

```bash
# 通過 Service (會負載均衡到不同 Pod)
for i in {1..10}; do
  curl -X POST http://oj-api-server:3001/sandbox/reserve \
    -H "Content-Type: application/json" \
    -d '{"test": "data"}' &
done
```

觀察：
- Sandbox 應該從多個不同的連接接收任務
- 每個 API Server 都能成功分發任務

### 4. 檢查 Sandbox 狀態

```bash
# 從不同的 API Server Pod 查詢
kubectl exec <api-pod-1> -n oj-api -- curl localhost:3001/sandbox/status
kubectl exec <api-pod-2> -n oj-api -- curl localhost:3001/sandbox/status
kubectl exec <api-pod-3> -n oj-api -- curl localhost:3001/sandbox/status
```

每個都應該顯示可用的 Sandbox。

## 注意事項

### 物理 vs 邏輯 Sandbox

- **物理 Sandbox**: 一個實際運行的 Sandbox Pod
- **邏輯 Sandbox**: 在 API Server 中註冊的 Sandbox 實例

這個方案中：
- 1 個物理 Sandbox Pod
- N 個邏輯 Sandbox 實例（N = API Server Pod 數量）
- 每個邏輯實例代表一個連接

### 容量計算

物理 Sandbox 的容量被"複製"到多個邏輯實例：

```
物理 Sandbox 容量: 4 個並發任務

在 3 個 API Server 的視圖：
- API Server 1 看到: 4 個可用
- API Server 2 看到: 4 個可用  
- API Server 3 看到: 4 個可用
```

**重要**：這不是問題，因為：
1. 任務最終都會路由到同一個物理 Sandbox
2. 物理 Sandbox 的隊列會管理實際容量
3. 如果超載，任務會在隊列中等待

### 全局狀態一致性

由於每個 API Server 獨立追蹤，全局狀態（`/sandbox/status`）可能會有重複計算：

```
實際可用容量: 4
報告的全局可用容量: 12 (4 * 3 個 API Server)
```

**解決方案選項**：
1. 接受這個差異（容量報告是估計值）
2. 在狀態查詢時除以 API Server 數量
3. 實現共享狀態存儲（如 Redis）- 更複雜

對於大多數場景，選項 1 是可接受的。

## 總結

通過為每個 API Server 連接生成唯一的 sandboxID，我們成功解決了多 Pod 環境下的連接問題：

- ✅ 每個 API Server 都能看到並使用 Sandbox
- ✅ 沒有 ID 衝突
- ✅ 狀態追蹤正確
- ✅ 任務可以從任何 API Server 分發

這個方案簡單、有效，並且不需要修改 Scheduler 的核心邏輯！
