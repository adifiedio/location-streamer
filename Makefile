.PHONY: network postgres createdb dropdb run-gateway run-tenant run-ingestion run-streamer docker-up docker-down sqlc migrateup migratedown migrateup1 migratedown1 new_migration test build build-tenant build-ingestion build-streamer

network:
	docker network create location-network

postgres:
	docker run --name postgres --network location-network -p 5432:5432 -e POSTGRES_USER=user -e POSTGRES_PASSWORD=password -d postgres:15-alpine

createdb:
	docker exec -it postgres createdb --username=user --owner=user location_db

dropdb:
	docker exec -it postgres dropdb location_db


build:
	@echo "building all services..."
	@make build-tenant
	@make build-ingestion
	@make build-streamer

build-tenant:
	go build -o bin/tenant cmd/tenant/main.go

build-ingestion:
	go build -o bin/ingestion cmd/ingestion/main.go

build-streamer:
	go build -o bin/streamer cmd/streamer/main.go

run-gateway:
	# gateway is currently implicit or can be implemented as a proxy
	# for now, we rely on accessing services directly via ports
	@echo "gateway logic is currently handled by direct service access or nginx (planned)"

run-tenant:
	APP_ENV=dev PORT=8081 go run cmd/tenant/main.go

run-ingestion:
	APP_ENV=dev INGESTION_PORT=8082 KAFKA_BROKER=localhost:9092 go run cmd/ingestion/main.go

run-streamer:
	APP_ENV=dev KAFKA_BROKER=localhost:9092 go run cmd/streamer/main.go

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

DB_URL=postgres://user:password@localhost:5432/location_db?sslmode=disable

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

sqlc:
	sqlc generate

test:
	go test -v ./...

