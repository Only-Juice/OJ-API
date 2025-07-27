build:
	swag init --parseDependency --parseInternal
	go build -o server main.go
	go build -o server-sandbox ./cmd/sandbox-server

run: build
	./server-sandbox & \
	./server & \
	wait

watch:
	reflex -s -R '^docs/' -r '\.go$$' make run 

clean:
	rm -r /sandbox/code/* /sandbox/repo/*