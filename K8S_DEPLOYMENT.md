# Kubernetes 部署指南

本文檔說明如何將 OJ-API 部署到 Kubernetes 集群。

## 目錄結構

```
k8s/
├── namespace.yaml           # 命名空間定義
├── configmap.yaml          # 配置映射（非敏感配置）
├── secret.yaml             # 機密配置（敏感資訊）
├── api-deployment.yaml     # API Server 部署
├── api-service.yaml        # API Server 服務（負載均衡）
├── api-service-headless.yaml # API Server Headless 服務（gRPC 連接）
├── sandbox-deployment.yaml # Sandbox Server 部署
├── ingress.yaml            # Ingress 配置（可選）
└── kustomization.yaml      # Kustomize 配置
```

## 前置需求

1. **Kubernetes 集群**（版本 >= 1.20）
2. **kubectl** 命令行工具
3. **Docker** 映像倉庫訪問權限
4. **PostgreSQL** 資料庫（可在集群內或外部）

### 可選組件

- **Ingress Controller**（如 Nginx Ingress Controller）- 用於外部訪問
- **cert-manager** - 用於自動生成 SSL/TLS 證書
- **kustomize** - 簡化配置管理（kubectl 1.14+ 已內建）

## 架構說明

### 多 API Server Pod 與 Sandbox 連接機制

本系統採用 **雙向 gRPC 流** 架構，支持多個 API Server Pod 同時運行：

```
                    ┌─────────────────────────────────────┐
                    │  Ingress / LoadBalancer (HTTP/HTTPS) │
                    └──────────────┬──────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────┐
                    │  oj-api-server Service (ClusterIP)  │
                    │  負載均衡 HTTP API 請求              │
                    └──────────────┬──────────────────────┘
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
   ┌────▼────┐                ┌────▼────┐                ┌────▼────┐
   │ API Pod │                │ API Pod │                │ API Pod │
   │    1    │                │    2    │                │    3    │
   └────▲────┘                └────▲────┘                └────▲────┘
        │                          │                          │
        └──────────────────────────┼──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────┐
                    │ oj-api-server-headless (Headless)   │
                    │ 返回所有 Pod IP（DNS A 記錄）        │
                    └──────────────▲──────────────────────┘
                                   │
                          gRPC 雙向流連接
                                   │
                    ┌──────────────┴──────────────────────┐
                    │                                     │
              ┌─────▼─────┐                       ┌──────▼──────┐
              │  Sandbox  │                       │  Sandbox    │
              │   Pod 1   │                       │   Pod 2     │
              └───────────┘                       └─────────────┘
```

#### 關鍵設計要點

1. **Headless Service 用於 gRPC 連接**
   - `oj-api-server-headless` 配置為 `ClusterIP: None`
   - DNS 查詢返回**所有** API Server Pod 的 IP 地址
   - Sandbox 通過 DNS 發現所有可用的 API Server Pod

2. **Sandbox 主動建立多個連接**
   - 每個 Sandbox Pod 定期（10秒）查詢 Headless Service 的 DNS
   - 為每個發現的 API Server Pod 建立獨立的 gRPC 雙向流連接
   - 自動處理 Pod 增減，動態維護連接池

3. **每個 API Server 獨立管理沙箱實例**
   - 每個 API Server Pod 運行獨立的 `SandboxScheduler`
   - Sandbox 連接到所有 API Server，所有調度器都能看到該 Sandbox
   - 任務可以從任何 API Server 分發到任何 Sandbox

4. **HTTP API 使用標準 Service**
   - `oj-api-server` Service 提供負載均衡
   - Ingress 或外部 LoadBalancer 將 HTTP 請求分發到任意 Pod

#### 優勢

- ✅ **高可用性**：任何 API Server Pod 都可以接收 HTTP 請求並分發任務
- ✅ **水平擴展**：可以隨時增加/減少 API Server Pod 數量
- ✅ **自動發現**：Sandbox 自動發現新的 API Server Pod
- ✅ **故障恢復**：Sandbox 自動重連斷開的連接
- ✅ **負載均衡**：所有 API Server 都可以向所有 Sandbox 分發任務

## 部署前準備

### 1. 構建 Docker 映像

```bash
# 構建 API Server 映像
docker build -t your-registry/oj-api:latest -f Dockerfile .

# 構建 Sandbox Server 映像
docker build -t your-registry/oj-sandbox:latest -f Dockerfile.sandbox .

# 推送到映像倉庫
docker push your-registry/oj-api:latest
docker push your-registry/oj-sandbox:latest
```

### 2. 更新配置

編輯 `k8s/configmap.yaml`，修改以下配置：

- `DB_HOST`: 資料庫主機地址
- `OJ_EXTERNAL_URL`: 外部訪問 URL
- `FRONTEND_URL`: 前端應用 URL
- `GIT_BASE_URL`: Gitea 基礎 URL
- `GITEA_URL`: Gitea 完整 URL

編輯 `k8s/secret.yaml`，修改以下敏感資訊：

- `DB_PASSWORD`: 資料庫密碼
- `JWT_SECRET`: JWT 密鑰
- `GITEA_CLIENT_ID`: Gitea OAuth Client ID
- `GITEA_CLIENT_SECRET`: Gitea OAuth Client Secret

**注意**: 在生產環境中，建議使用更安全的方式管理 Secret，例如：
- 使用 `kubectl create secret` 命令創建
- 使用 Sealed Secrets
- 使用外部密鑰管理服務（如 HashiCorp Vault）

### 3. 更新映像倉庫地址

編輯 `k8s/api-deployment.yaml` 和 `k8s/sandbox-deployment.yaml`，將 `your-registry` 替換為您的實際映像倉庫地址。

或者編輯 `k8s/kustomization.yaml` 中的映像配置：

```yaml
images:
  - name: your-registry/oj-api
    newName: your-actual-registry/oj-api
    newTag: v1.0.0
  - name: your-registry/oj-sandbox
    newName: your-actual-registry/oj-sandbox
    newTag: v1.0.0
```

### 4. 配置 Ingress（可選）

如果需要外部訪問，編輯 `k8s/ingress.yaml`：

- 修改 `host` 為您的域名
- 配置 TLS 證書（如果使用 HTTPS）

## 部署步驟

### 方法一：使用 kubectl 直接部署

```bash
# 進入 k8s 目錄
cd k8s

# 創建命名空間
kubectl apply -f namespace.yaml

# 創建 ConfigMap 和 Secret
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml

# 部署 API Server
kubectl apply -f api-deployment.yaml
kubectl apply -f api-service.yaml
kubectl apply -f api-service-headless.yaml

# 部署 Sandbox Server
kubectl apply -f sandbox-deployment.yaml

# 部署 Ingress（可選）
kubectl apply -f ingress.yaml
```

### 方法二：使用 Kustomize 部署

```bash
# 從項目根目錄執行
kubectl apply -k k8s/

# 或者使用 kustomize 工具
kustomize build k8s/ | kubectl apply -f -
```

## 驗證部署

### 檢查 Pod 狀態

```bash
# 查看所有 Pod
kubectl get pods -n oj-api

# 查看 Pod 詳細信息
kubectl describe pod <pod-name> -n oj-api

# 查看 Pod 日誌
kubectl logs <pod-name> -n oj-api

# 實時跟蹤日誌
kubectl logs -f <pod-name> -n oj-api
```

### 檢查服務狀態

```bash
# 查看所有服務
kubectl get svc -n oj-api

# 查看 Ingress（如果配置了）
kubectl get ingress -n oj-api
```

### 測試 API

```bash
# 使用 port-forward 測試（不使用 Ingress）
kubectl port-forward -n oj-api svc/oj-api-server 3001:3001

# 在另一個終端測試
curl http://localhost:3001/health
```

## 常見問題排查

### Pod 無法啟動

```bash
# 查看 Pod 事件
kubectl describe pod <pod-name> -n oj-api

# 查看日誌
kubectl logs <pod-name> -n oj-api

# 檢查資源配額
kubectl top pods -n oj-api
```

### 映像拉取失敗

如果使用私有倉庫，創建 ImagePullSecret：

```bash
kubectl create secret docker-registry registry-secret \
  --docker-server=<your-registry-server> \
  --docker-username=<your-username> \
  --docker-password=<your-password> \
  --docker-email=<your-email> \
  -n oj-api
```

然後在 Deployment 中添加：

```yaml
spec:
  template:
    spec:
      imagePullSecrets:
      - name: registry-secret
```

### Sandbox 特權模式問題

Sandbox Server 需要特權模式來運行 isolate。某些 Kubernetes 集群可能限制特權容器。

解決方案：
1. 確認集群支持特權容器
2. 使用 PodSecurityPolicy 或 PodSecurityStandard 允許特權容器
3. 或者使用其他沙箱解決方案

### 數據庫連接問題

確認：
1. 資料庫主機地址可從集群內訪問
2. 資料庫防火牆規則允許 Pod 訪問
3. ConfigMap 和 Secret 中的資料庫配置正確

## 更新部署

### 更新映像版本

```bash
# 方法一：直接更新映像
kubectl set image deployment/oj-api-server oj-api=your-registry/oj-api:v1.1.0 -n oj-api
kubectl set image deployment/oj-sandbox-server sandbox=your-registry/oj-sandbox:v1.1.0 -n oj-api

# 方法二：編輯 Deployment
kubectl edit deployment oj-api-server -n oj-api

# 方法三：使用 Kustomize 更新
# 編輯 kustomization.yaml 中的映像標籤，然後重新應用
kubectl apply -k k8s/
```

### 更新配置

```bash
# 更新 ConfigMap
kubectl apply -f k8s/configmap.yaml

# 更新 Secret
kubectl apply -f k8s/secret.yaml

# 重啟 Pod 以應用新配置
kubectl rollout restart deployment/oj-api-server -n oj-api
kubectl rollout restart deployment/oj-sandbox-server -n oj-api
```

### 回滾部署

```bash
# 查看部署歷史
kubectl rollout history deployment/oj-api-server -n oj-api

# 回滾到上一個版本
kubectl rollout undo deployment/oj-api-server -n oj-api

# 回滾到特定版本
kubectl rollout undo deployment/oj-api-server --to-revision=2 -n oj-api
```

## 擴展與伸縮

### 手動擴展

```bash
# 擴展 API Server 副本數（Sandbox 會自動連接到新的 Pod）
kubectl scale deployment/oj-api-server --replicas=3 -n oj-api

# 擴展 Sandbox Server 副本數
kubectl scale deployment/oj-sandbox-server --replicas=2 -n oj-api
```

**重要提示**：擴展 API Server 時，Sandbox 會在 10 秒內自動發現並連接到新的 Pod，無需重啟 Sandbox。

### 驗證多 Pod 連接

擴展 API Server 後，可以檢查 Sandbox 日誌確認連接：

```bash
# 查看 Sandbox 日誌
kubectl logs -f <sandbox-pod-name> -n oj-api

# 應該看到類似以下的日誌：
# Connecting to new scheduler at 10.244.1.5:3001
# Connected to scheduler at 10.244.1.5:3001
# Managing connections to 3 scheduler(s): [10.244.1.5:3001 10.244.1.6:3001 10.244.1.7:3001]
```

### 自動擴展（HPA）

創建 HPA 配置：

```bash
# API Server 自動擴展
kubectl autoscale deployment oj-api-server \
  --cpu-percent=70 \
  --min=2 \
  --max=10 \
  -n oj-api

# Sandbox Server 自動擴展
kubectl autoscale deployment oj-sandbox-server \
  --cpu-percent=80 \
  --min=1 \
  --max=5 \
  -n oj-api
```

## 監控與日誌

### 查看資源使用

```bash
# 查看 Pod 資源使用
kubectl top pods -n oj-api

# 查看 Node 資源使用
kubectl top nodes
```

### 集中式日誌

建議使用以下工具進行日誌管理：
- **ELK Stack** (Elasticsearch, Logstash, Kibana)
- **Loki** + Grafana
- **Fluentd** + Elasticsearch

### 監控

建議使用以下工具進行監控：
- **Prometheus** + Grafana
- **Datadog**
- **New Relic**

## 清理資源

```bash
# 刪除所有資源
kubectl delete -k k8s/

# 或者逐個刪除
kubectl delete namespace oj-api
```

## 安全建議

1. **不要在 Git 中提交真實的 Secret**
   - 使用 `.gitignore` 忽略包含敏感資訊的文件
   - 使用環境變數或外部密鑰管理

2. **使用 RBAC 限制訪問權限**
   - 為服務賬戶配置最小權限
   - 定期審查權限

3. **啟用 Network Policy**
   - 限制 Pod 之間的網絡訪問
   - 只允許必要的通信

4. **定期更新映像**
   - 修補安全漏洞
   - 使用映像掃描工具

5. **使用 TLS/SSL**
   - 為 Ingress 配置 HTTPS
   - 使用 cert-manager 自動管理證書

## 生產環境建議

1. **資源配額**
   - 為命名空間設置資源配額
   - 防止資源耗盡

2. **備份策略**
   - 定期備份資料庫
   - 備份重要配置

3. **災難恢復計劃**
   - 準備恢復流程
   - 定期演練

4. **多環境管理**
   - 使用 Kustomize overlays 管理不同環境
   - 創建 dev、staging、production 環境

5. **CI/CD 集成**
   - 自動化構建和部署流程
   - 使用 GitOps 工具（如 ArgoCD、Flux）

## 參考資源

- [Kubernetes 官方文檔](https://kubernetes.io/docs/)
- [Kustomize 文檔](https://kustomize.io/)
- [kubectl 命令參考](https://kubernetes.io/docs/reference/kubectl/)
- [Kubernetes 最佳實踐](https://kubernetes.io/docs/concepts/configuration/overview/)

## 支援

如有問題或需要幫助，請：
1. 查看日誌和錯誤信息
2. 參考本文檔的故障排查部分
3. 聯繫開發團隊或提交 Issue
