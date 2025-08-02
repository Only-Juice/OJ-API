build:
	swag init --parseDependency --parseInternal
	go build -o server main.go
	go build -o server-sandbox ./cmd/sandbox-server
	cd ./sandbox/grp_parser && go build -o grp_parser main.go

run: build
	./server

run-sandbox: build
	./server-sandbox

run-all: build
	./server & \
	sleep 2 && \
	./server-sandbox

watch:
	reflex -s -R '^docs/' -r '\.go$$' make run-all

clean:
	rm -r /sandbox/code/* /sandbox/repo/*

proto:
	./generate_proto.sh

test-grpc:
	go run cmd/test-grpc/main.go

test-scheduler:
	go run cmd/test-scheduler/main.go

.PHONY: build run run-sandbox run-all watch clean proto test-grpc test-scheduler