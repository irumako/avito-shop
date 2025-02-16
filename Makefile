integration_test:
	docker compose up -d --build
	go test -v -tags=integration ./...
	docker compose down -v --remove-orphans

build: integration_test
	docker compose up -d --build

stop:
	docker compose down