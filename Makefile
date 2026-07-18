.PHONY: chat test-chat test-chat-race test-rooms-race db-up db-down db-reset db-shell db-users db-messages

# ---------- app ----------
chat:
	@echo "Running the chat server..."
	@go mod tidy
	@go run .

# ---------- database ----------
db-up:
	@echo "Starting PostgreSQL (container: go-websocket-db, host port 5433)..."
	@docker compose up -d

db-down:
	@echo "Stopping PostgreSQL (data is kept in the volume)..."
	@docker compose down

db-reset:
	@echo "Stopping PostgreSQL and DELETING all data..."
	@docker compose down -v

db-shell:
	@docker exec -it go-websocket-db psql -U chat -d chatdb

db-users:
	@docker exec go-websocket-db psql -U chat -d chatdb -c "SELECT id, username, created_at FROM users ORDER BY id;"

db-messages:
	@docker exec go-websocket-db psql -U chat -d chatdb -c "SELECT id, sender_name, recipient_name, room_id, data, created_at FROM messages ORDER BY id DESC LIMIT 20;"

# ---------- tests ----------
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
