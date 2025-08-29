import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定義指標
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const successfulRequests = new Counter('successful_requests');

// 測試配置 - 可根據需求調整
export let options = {
  stages: [
    { duration: '30s', target: 5 },   // 逐漸增加到5個用戶
    { duration: '1m', target: 10 },   // 保持10個用戶1分鐘
    { duration: '30s', target: 0 },   // 逐漸降到0
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95%的請求在500ms內完成
    http_req_failed: ['rate<0.1'],    // 錯誤率低於10%
    errors: ['rate<0.05'],            // 自定義錯誤率低於5%
  },
  insecureSkipTLSVerify: true,
};

// 配置常數
const BASE_URL = 'https://oj.zre.tw:3001/api/gitea';
const SLEEP_DURATION = Math.random() * 2 + 1; // 隨機1-3秒

// 生成動態測試數據
function generateWebhookPayload() {
  const timestamp = new Date().toISOString();
  const commitId = "0000000000000000000000000000000000000000";
  
  return JSON.stringify({
    ref: "refs/heads/main",
    before: "0000000000000000000000000000000000000000",
    after: commitId,
    compare_url: `https://oj.zre.tw:3000/username/OOP2024f_HW3/compare/0000000000000000000000000000000000000000...${commitId}`,
    commits: [
      {
        id: commitId,
        message: `Test commit ${Math.floor(Math.random() * 1000)}`,
        url: `https://oj.zre.tw:3000/username/OOP2024f_HW3/commit/${commitId}`,
        author: {
          name: "Test User",
          email: "test@example.com",
          username: "testuser"
        },
        committer: {
          name: "Test User",
          email: "test@example.com",
          username: "testuser"
        },
        verification: null,
        timestamp: timestamp,
        added: ["test.txt"],
        removed: null,
        modified: null
      }
    ],
    total_commits: 1,
    head_commit: {
      id: commitId,
      message: `Test commit ${Math.floor(Math.random() * 1000)}`,
      url: `https://oj.zre.tw:3000/username/OOP2024f_HW3/commit/${commitId}`,
      author: {
        name: "Test User",
        email: "test@example.com",
        username: "testuser"
      },
      committer: {
        name: "Test User",
        email: "test@example.com",
        username: "testuser"
      },
      verification: null,
      timestamp: timestamp,
      added: ["test.txt"],
      removed: null,
      modified: null
    },
    repository: {
      id: 10,
      owner: {
        id: 3,
        login: "username",
        full_name: "Test User",
        email: "username@noreply.localhost",
        html_url: "https://oj.zre.tw:3000/username",
        username: "username"
      },
      name: "OOP2024f_HW3",
      full_name: "username/OOP2024f_HW3",
      description: "Test repository",
      private: true,
      html_url: "https://oj.zre.tw:3000/username/OOP2024f_HW3",
      clone_url: "https://oj.zre.tw:3000/username/OOP2024f_HW3.git",
      default_branch: "main",
      created_at: timestamp,
      updated_at: timestamp
    },
    pusher: {
      id: 3,
      login: "username",
      email: "username@noreply.localhost",
      username: "username"
    },
    sender: {
      id: 3,
      login: "username",
      email: "username@noreply.localhost",
      username: "username"
    }
  });
}

// 生成動態請求標頭
function generateHeaders() {
  const deliveryId = Math.random().toString(36).substring(2, 15) + 
                    Math.random().toString(36).substring(2, 15);
  
  return {
    'Content-Type': 'application/json',
    'User-Agent': 'Gitea/1.0',
    'X-GitHub-Delivery': deliveryId,
    'X-GitHub-Event': 'push',
    'X-GitHub-Event-Type': 'push',
    'X-Gitea-Delivery': deliveryId,
    'X-Gitea-Event': 'push',
    'X-Gitea-Event-Type': 'push',
    'X-Hub-Signature': 'sha1=test',
    'X-Hub-Signature-256': 'sha256=test',
  };
}

// 主要測試函數
export default function () {
  group('Gitea Webhook Test', function () {
    const payload = generateWebhookPayload();
    const headers = generateHeaders();
    
    const startTime = Date.now();
    const response = http.post(BASE_URL, payload, { 
      headers,
      timeout: '30s', // 設定超時時間
      tags: { test_type: 'webhook' }
    });
    const endTime = Date.now();
    
    // 記錄自定義指標
    responseTime.add(endTime - startTime);
    
    // 詳細的檢查
    const isSuccess = check(response, {
      '狀態碼是 200': (r) => r.status === 200,
      '回應時間 < 1000ms': (r) => r.timings.duration < 1000,
      '回應內容不為空': (r) => r.body && r.body.length > 0,
      '內容類型正確': (r) => r.headers['Content-Type'] && 
                            r.headers['Content-Type'].includes('application/json'),
    });
    
    // 記錄結果
    if (isSuccess) {
      successfulRequests.add(1);
    } else {
      errorRate.add(1);
      console.error(`請求失敗: ${response.status} - ${response.body}`);
    }
    
    // 詳細日志記錄
    console.log(`[VU: ${__VU}, Iter: ${__ITER}] POST ${BASE_URL} -> ${response.status} (${response.timings.duration.toFixed(2)}ms)`);
    
    // 在錯誤情況下記錄更多信息
    if (response.status !== 200) {
      console.error(`錯誤詳情: ${JSON.stringify({
        status: response.status,
        body: response.body ? response.body.slice(0, 500) : 'No body',
        headers: response.headers,
      }, null, 2)}`);
    }
  });
  
  // 隨機睡眠時間，模擬真實使用情況
  sleep(SLEEP_DURATION);
}

// 測試設置階段
export function setup() {
  console.log('開始 Gitea Webhook 負載測試...');
  return { startTime: Date.now() };
}

// 測試清理階段
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`測試完成，總耗時: ${duration.toFixed(2)}秒`);
}