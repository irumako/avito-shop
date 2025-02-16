unit_test:
	go test -v ./cmd/api -cover
integration_test:
	docker compose up -d --build
	go test -v -tags=integration ./...
	docker compose down -v --remove-orphans

build: unit_test integration_test
	docker compose up -d --build

stop:
	docker compose down