#!/bin/bash

# 安裝必要工具
echo "Installing protoc and grpc tools..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# 創建輸出目錄
mkdir -p proto

# 生成 protobuf 和 gRPC 代碼
echo "Generating protobuf and gRPC code..."
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/sandbox.proto

echo "Done!"
