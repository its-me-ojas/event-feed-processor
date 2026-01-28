# Simple Makefile for managing the Event-Driven Feed System

.PHONY: infra run-api run-processor test clean

infra:
	docker-compose up -d

stop-infra:
	docker-compose down

run-api:
	go run cmd/api/main.go

run-processor:
	go run cmd/processor/main.go

test:
	go run cmd/e2e-test/main.go

dlq:
	go run cmd/dlq-inspector/main.go

clean:
	rm -f api processor dlq-inspector e2e-test
