APP_NAME=dockdns

run:
	go run main.go

build:
	go build -o bin/$(APP_NAME) main.go 