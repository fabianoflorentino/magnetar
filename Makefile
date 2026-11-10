.PHONY: build run test docker docker-up docker-down clean

build:
	go build -o bin/magnetar ./cmd/magnetar

run:
	go run ./cmd/magnetar

test:
	go test ./... -v

docker:
	docker build -t magnetar .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

clean:
	rm -rf bin/
