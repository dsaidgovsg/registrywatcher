all: fmt test

test:
	@echo "-- running tests --"
	go test ./...
	@echo "-- tests done --\n\n"

fmt:
	@echo "-- running formatter --"
	go fmt ./...
	@echo "-- formatter done --\n\n"