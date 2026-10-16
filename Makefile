.PHONY: up down logs lint lint-restaurant test

up:
	docker compose up --build

down:
	docker compose down -v

logs:
	docker compose logs -f api restaurant-service

lint:
	golangci-lint run ./...

lint-restaurant:
	cd restaurant-service && golangci-lint run ./...

test:
	go test ./...
	cd restaurant-service && go test ./...
