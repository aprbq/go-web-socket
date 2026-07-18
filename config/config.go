package config

import (
	"os"

	"github.com/joho/godotenv"
)

var (
	WSPort    = ":3223"
	DBDsn     = "host=localhost user=chat password=chatpass dbname=chatdb port=5433 sslmode=disable"
	JWTSecret = "dev-secret"
)

// Load reads .env (if present) and env vars, overriding the defaults above.
func Load() {
	godotenv.Load()

	if v := os.Getenv("WS_PORT"); v != "" {
		WSPort = v
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		DBDsn = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		JWTSecret = v
	}
}
