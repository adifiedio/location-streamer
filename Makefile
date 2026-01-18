.PHONY: run-gateway run-tenant run-ingestion run-streamer docker-up docker-down sqlc

run-gateway:
	go run cmd/gateway/main.go

run-tenant:
	go run cmd/tenant/main.go

run-ingestion:
	go run cmd/ingestion/main.go

run-streamer:
	go run cmd/streamer/main.go

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

sqlc:
	sqlc generate

test:
	go test -v ./...
