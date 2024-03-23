all: fmt lint test

test:
	@echo "===================================="
	@echo "########## Running tests ##########"
	@echo "===================================="
	@ENV=test go test ./...
	@echo "================= OK ================="
	@echo ""

fmt:
	@echo "===================================="
	@echo "######### Formatting code ##########"
	@echo "===================================="
	@goimports-reviser -rm-unused -imports-order "std,company,project,general" ./...
	@gofmt -l -w .
	@echo "================= OK ================="
	@echo ""

fmt-check:
	@echo "===================================="
	@echo "####### Checking Formatting ########"
	@echo "===================================="
	@goimports-reviser -list-diff -imports-order "std,company,project,general" ./...
	@files=$(gofmt -l .) && [ -z "$files" ]
	@echo "================= OK ================="
	@echo ""

lint:
	@echo "===================================="
	@echo "########## Linting code ###########"
	@echo "===================================="
	@golangci-lint run
	@echo "================= OK ================="
	@echo ""

test-env:
	./env/test/script.sh

dev:
	docker compose -f env/dev/docker-compose.yml up --build