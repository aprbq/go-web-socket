.PHONY: chat test-chat test-chat-race test-rooms-race

chat:
	@echo "Running the chat server..."
	@go mod tidy
	@go run .

test-chat:
	@echo "Running the tests..."
	@go clean -testcache
	@go test -v ./...

test-chat-race:
	@echo "Running the tests with race detector..."
	@go clean -testcache
	@go test -race -v ./...

test-rooms-race:
	@echo "Running the rooms tests with race detector..."
	@go clean -testcache
	@go test -race -v -timeout 30s -run TestRooms .
