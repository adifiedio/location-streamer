.PHONY: network postgres createdb dropdb run-gateway run-tenant run-ingestion run-streamer docker-up docker-down sqlc migrateup migratedown migrateup1 migratedown1 new_migration test

network:
	docker network create location-network

postgres:
	docker run --name postgres --network location-network -p 5432:5432 -e POSTGRES_USER=user -e POSTGRES_PASSWORD=password -d postgres:15-alpine

createdb:
	docker exec -it postgres createdb --username=user --owner=user location_db

dropdb:
	docker exec -it postgres dropdb location_db


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

