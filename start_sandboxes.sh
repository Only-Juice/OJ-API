#!/bin/bash

# 啟動多個沙箱服務器的腳本

# 設置環境變數
export SANDBOX_COUNT=2
export SCHEDULER_ADDRESS=localhost:50052

# 啟動第一個沙箱服務器
echo "Starting sandbox server 1..."
./server-sandbox &
SANDBOX1_PID=$!

# 等待一秒
sleep 1

# 啟動第二個沙箱服務器  
echo "Starting sandbox server 2..."
./server-sandbox &
SANDBOX2_PID=$!

# 等待一秒
sleep 1

# 啟動第三個沙箱服務器
echo "Starting sandbox server 3..."
./server-sandbox &
SANDBOX3_PID=$!

echo "All sandbox servers started."
echo "Sandbox 1 PID: $SANDBOX1_PID"
echo "Sandbox 2 PID: $SANDBOX2_PID"
echo "Sandbox 3 PID: $SANDBOX3_PID"

# 創建停止腳本
cat > stop_sandboxes.sh << 'EOF'
#!/bin/bash
echo "Stopping all sandbox servers..."
pkill -f "server-sandbox"
echo "All sandbox servers stopped."
EOF

chmod +x stop_sandboxes.sh

echo "Created stop_sandboxes.sh to stop all sandbox servers."
echo "Press Ctrl+C to stop all servers or run ./stop_sandboxes.sh"

# 等待信號
trap "echo 'Stopping all sandbox servers...'; kill $SANDBOX1_PID $SANDBOX2_PID $SANDBOX3_PID 2>/dev/null; exit" INT TERM

# 等待所有後台進程
wait
