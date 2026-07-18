# go-web-socket

![race-tests](https://github.com/aprbq/go-web-socket/actions/workflows/test.yml/badge.svg)

Real-time private chat server in Go — gorilla/websocket + JWT auth + PostgreSQL,
structured with hexagonal architecture (`handlers` → `services` → `repositories`).

## Features

- Register / login with bcrypt password hashing and JWT (24h)
- Direct messages over WebSocket (`/ws?token=...`) — delivered to every open tab
  of both users, and persisted even when the recipient is offline
- REST endpoints: user search, DM history, conversation list (all bearer-token protected)
- A single `AcceptLoop` goroutine owns all connection state — no locks,
  race-free by design (verified under the Go race detector)

## Run

```
make db-up    # start PostgreSQL in docker (host port 5433)
make chat     # start the server on :3223
```

Then open `chat.html` in a browser — open two windows to chat between two users.

## Test

```
make test-chat        # full suite on in-memory mock repositories (no DB needed)
make test-chat-race   # same suite under the Go race detector
```

The race suite goes beyond a 2-user happy path: a 20-user × 50-message
concurrent ring test (1,000 messages in flight) and a join/leave churn test
that mutates the clients map while deliveries iterate it. Every push runs
`make test-chat-race` in GitHub Actions — the badge above shows the latest result.

## Useful targets

```
make db-users      # list registered users
make db-messages   # show the latest 20 messages
make db-shell      # open psql inside the container
make db-down       # stop PostgreSQL (data kept)
make db-reset      # stop PostgreSQL and DELETE all data
```
