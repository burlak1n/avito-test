.PHONY: build run stop clean test

build:
	docker-compose build

run:
	docker-compose up

stop:
	docker-compose down

clean:
	docker-compose down -v

test:
	go test ./...

lint:
	golangci-lint run

