# Kubernetes 多 API Server Pod 架構說明

## 問題背景

在原始設計中，Sandbox 通過 gRPC 雙向流連接到 API Server。但在 Kubernetes 環境中，當部署多個 API Server Pod 時會遇到以下問題：

- **問題**：Sandbox 通過 Service 連接時，Service 會做負載均衡
- **影響**：gRPC 雙向流是長連接，Sandbox 可能只連接到一個 API Server Pod
- **後果**：其他 API Server Pod 無法看到該 Sandbox，導致任務分發不均或失敗

## 解決方案

採用 **Headless Service + 多連接** 架構：

### 1. Headless Service

創建 `oj-api-server-headless` Service：
- 設置 `clusterIP: None`
- DNS 查詢返回**所有** Pod IP，而非單一 ClusterIP
- Sandbox 可以發現所有可用的 API Server Pod

### 2. Sandbox 多連接管理

修改 Sandbox 連接邏輯：
- 定期（10秒）通過 DNS 解析 Headless Service
- 為每個發現的 API Server Pod IP 建立獨立的 gRPC 連接
- **關鍵**：為每個連接生成唯一的 sandboxID（格式：`{base-uuid}-{host-ip-port}`）
- 自動處理 Pod 增減，動態維護連接池
- 連接斷開時自動重連

**為什麼需要唯一的 sandboxID？**

每個 API Server 的 Scheduler 維護自己的 `instances map[string]*SandboxInstance`。如果多個連接使用相同的 sandboxID，會導致：
- ID 衝突，連接互相覆蓋
- 只有一個 API Server 能看到 Sandbox

**解決方案**：
```go
// 基礎 ID
baseSandboxID := uuid.New().String()  // 例如: 550e8400-e29b-41d4-a716-446655440000

// 為每個連接生成唯一 ID
uniqueSandboxID := fmt.Sprintf("%s-%s", baseSandboxID, sanitizeHostAddr(hostAddr))

// 例如:
// 連接 1: 550e8400-...-10-244-1-5-3001
// 連接 2: 550e8400-...-10-244-1-6-3001
// 連接 3: 550e8400-...-10-244-1-7-3001
```

這樣每個 API Server 都能正確追蹤各自的 Sandbox 連接。

## 架構圖

```
┌─────────────────────────────────────┐
│  HTTP 流量 (通過標準 Service)        │
│  oj-api-server:3001                 │
└───────────┬─────────────────────────┘
            │ (負載均衡)
            ▼
   ┌────────┴────────┬──────────┐
   │                 │          │
┌──▼───┐      ┌──────▼─┐    ┌──▼───┐
│ API  │      │  API   │    │ API  │
│ Pod1 │      │  Pod2  │    │ Pod3 │
└──▲───┘      └──▲─────┘    └──▲───┘
   │             │             │
   └─────────────┼─────────────┘
                 │
┌────────────────▼────────────────────┐
│ gRPC 流量 (通過 Headless Service)   │
│ oj-api-server-headless:3001         │
│ DNS 返回: [Pod1_IP, Pod2_IP, Pod3_IP]│
└────────────────▲────────────────────┘
                 │
            (多個連接)
                 │
        ┌────────┴────────┐
        │                 │
   ┌────▼────┐      ┌─────▼────┐
   │ Sandbox │      │ Sandbox  │
   │  Pod1   │      │   Pod2   │
   └─────────┘      └──────────┘
```

## 關鍵文件變更

### 1. 新增 Headless Service

**文件**: `k8s/api-service-headless.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: oj-api-server-headless
  namespace: oj-api
spec:
  type: ClusterIP
  clusterIP: None  # Headless Service
  ports:
  - port: 3001
    targetPort: 3001
    protocol: TCP
    name: grpc-http
  selector:
    app: oj-api-server
```

### 2. 更新 ConfigMap

**文件**: `k8s/configmap.yaml`

```yaml
data:
  # 從標準 Service 改為 Headless Service
  SCHEDULER_ADDRESS: "oj-api-server-headless:3001"
```

### 3. 修改 Sandbox 連接邏輯

**文件**: `cmd/sandbox-server/main.go`

新增功能：
- `manageMultipleConnections()`: 管理多個連接
- `connectToAllSchedulers()`: 解析 DNS 並建立連接
- `maintainConnection()`: 維護單個連接並自動重連

核心邏輯：
```go
// 解析 DNS 獲取所有 IP
ips, err := net.LookupIP(host)

// 為每個 IP 建立連接
for _, ip := range ips {
    hostAddr := net.JoinHostPort(ip.String(), port)
    go maintainConnection(ctx, hostAddr, sandboxID, sandboxInstance)
}
```

### 4. 更新部署文檔

**文件**: 
- `K8S_DEPLOYMENT.md`: 詳細部署說明和架構圖
- `GRPC_ARCHITECTURE.md`: gRPC 架構和 K8s 多 Pod 支持
- `k8s/kustomization.yaml`: 包含 headless service

## 工作原理

### DNS 發現機制

1. Sandbox 查詢 `oj-api-server-headless.oj-api.svc.cluster.local`
2. DNS 返回所有健康 Pod 的 IP 列表：
   ```
   10.244.1.5
   10.244.1.6
   10.244.1.7
   ```
3. Sandbox 為每個 IP 建立獨立的 gRPC 雙向流連接

### 動態擴展

當 API Server 擴展時：

```bash
# 從 1 個擴展到 3 個
kubectl scale deployment/oj-api-server --replicas=3 -n oj-api
```

**自動發生**：
1. 新的 API Server Pod 啟動
2. Headless Service DNS 記錄自動更新
3. Sandbox 在下次 DNS 查詢（10秒內）發現新 Pod
4. Sandbox 自動建立到新 Pod 的連接

**日誌輸出**：
```
Connecting to new scheduler at 10.244.1.7:3001
Connected to scheduler at 10.244.1.7:3001
Managing connections to 3 scheduler(s): [10.244.1.5:3001 10.244.1.6:3001 10.244.1.7:3001]
```

### 故障恢復

當 API Server Pod 故障時：

1. gRPC 連接斷開（自動檢測）
2. DNS 記錄自動更新（移除故障 Pod）
3. Sandbox 停止重連已移除的 Pod
4. 維持與健康 Pod 的連接

## 優勢

### ✅ 高可用性
- 任何 API Server Pod 都可以接收 HTTP 請求
- 任何 API Server Pod 都可以看到並使用所有 Sandbox
- 單個 Pod 故障不影響其他連接

### ✅ 水平擴展
- 可以隨時增加/減少 API Server Pod 數量
- Sandbox 自動適應 Pod 數量變化
- 無需手動配置或重啟

### ✅ 負載均衡
- HTTP 請求通過標準 Service 負載均衡
- gRPC 連接分散到所有 Pod
- 所有 API Server 都能向所有 Sandbox 分發任務

### ✅ 零停機更新
- 滾動更新期間保持服務可用
- 新 Pod 自動接管連接
- 舊 Pod 優雅關閉

## 部署驗證

### 1. 部署資源

```bash
# 應用所有配置
kubectl apply -k k8s/

# 確認 Headless Service
kubectl get svc -n oj-api
# 應該看到 oj-api-server-headless，CLUSTER-IP 為 None
```

### 2. 擴展 API Server

```bash
# 擴展到 3 個副本
kubectl scale deployment/oj-api-server --replicas=3 -n oj-api

# 等待 Pod 啟動
kubectl get pods -n oj-api -w
```

### 3. 檢查 DNS

```bash
# 進入 Sandbox Pod
kubectl exec -it <sandbox-pod-name> -n oj-api -- sh

# 查詢 DNS（需要 nslookup 或 dig）
nslookup oj-api-server-headless.oj-api.svc.cluster.local

# 應該返回多個 IP 地址
```

### 4. 驗證連接

```bash
```bash
# 查看 Sandbox 日誌
kubectl logs -f <sandbox-pod-name> -n oj-api

# 應該看到類似以下的日誌：
# Sandbox base ID: 550e8400-e29b-41d4-a716-446655440000
# Connecting to new scheduler at 10.244.1.5:3001
# Connected to scheduler at 10.244.1.5:3001
# Connecting to new scheduler at 10.244.1.6:3001
# Connected to scheduler at 10.244.1.6:3001
# Managing connections to 3 scheduler(s): [10.244.1.5:3001 10.244.1.6:3001 10.244.1.7:3001]

# 驗證每個 API Server 都能看到 Sandbox
kubectl logs <api-pod-1> -n oj-api | grep "Sandbox.*connected"
# 應該輸出: Sandbox 550e8400-...-10-244-1-5-3001 connected successfully

kubectl logs <api-pod-2> -n oj-api | grep "Sandbox.*connected"  
# 應該輸出: Sandbox 550e8400-...-10-244-1-6-3001 connected successfully
```
```

### 5. 測試故障恢復

```bash
# 刪除一個 API Server Pod
kubectl delete pod <api-pod-name> -n oj-api

# 觀察 Sandbox 日誌
kubectl logs -f <sandbox-pod-name> -n oj-api

# 應該看到連接斷開和自動移除該 Pod
```

## 性能考量

### DNS 查詢頻率
- 默認：每 10 秒查詢一次
- 可調整以平衡反應速度和 DNS 負載

### 連接數
- 每個 Sandbox Pod 建立 N 個連接（N = API Server Pod 數量）
- 示例：3 個 API Server + 2 個 Sandbox = 6 個 gRPC 連接
- gRPC 連接開銷小，可支持較多連接

### 內存使用
- 每個額外連接增加少量內存（約 1-2 MB）
- 建議 API Server Pod 數量 < 10

## 故障排查

### Sandbox 無法連接

```bash
# 檢查 Headless Service
kubectl get svc oj-api-server-headless -n oj-api

# 檢查 DNS
kubectl run -it --rm debug --image=busybox --restart=Never -n oj-api -- nslookup oj-api-server-headless

# 檢查 Sandbox 日誌
kubectl logs <sandbox-pod-name> -n oj-api | grep "Failed to connect"
```

### 連接數不匹配

```bash
# 檢查 API Server Pod 數量
kubectl get pods -n oj-api -l app=oj-api-server

# 檢查 Sandbox 管理的連接數
kubectl logs <sandbox-pod-name> -n oj-api | grep "Managing connections"
```

### DNS 解析失敗

可能原因：
1. CoreDNS 未正常運行
2. Service 未正確創建
3. Pod 未就緒（如果啟用了 `publishNotReadyAddresses: false`）

## 未來改進

### 可能的優化

1. **連接池大小限制**
   - 限制單個 Sandbox 的最大連接數
   - 適用於超大規模部署

2. **連接健康檢查**
   - 主動檢測連接健康狀態
   - 更快發現和恢復故障連接

3. **基於區域的連接**
   - 優先連接同一可用區的 Pod
   - 降低跨區域延遲

4. **指標收集**
   - 暴露連接數、重連次數等指標
   - 集成 Prometheus 監控

## 總結

通過 Headless Service + 多連接架構，我們成功解決了 K8s 環境中多 API Server Pod 的連接問題：

- ✅ **自動發現**：Sandbox 自動發現所有 API Server Pod
- ✅ **動態擴展**：支持 API Server 水平擴展
- ✅ **高可用**：單個 Pod 故障不影響整體服務
- ✅ **零配置**：Pod 增減無需手動干預

這種架構為系統在 Kubernetes 中的水平擴展和高可用性奠定了基礎。
