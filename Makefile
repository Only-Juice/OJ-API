build:
	swag init --parseDependency --parseInternal
	go build -o server main.go

run: build
	./server

watch:
	reflex -s -R '^docs/' -r '\.go$$' make run
