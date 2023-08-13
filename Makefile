server:
	go build -o bin/server cmd/main.go
	./bin/server

db:
	docker-compose up -d
