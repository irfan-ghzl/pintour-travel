package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Fonnte   FonnteConfig
	Email    EmailConfig
	Supabase SupabaseConfig
}

type EmailConfig struct {
	ResendAPIKey string
	FromAddress  string
	AppURL       string // used for reset password link
}

type SupabaseConfig struct {
	URL        string
	ServiceKey string
}

type ServerConfig struct {
	Port          string
	Env           string
	PortalBaseURL string
}

type FonnteConfig struct {
	APIToken string
}

type DatabaseConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret          string
	ExpirationHours int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:          getEnv("SERVER_PORT", "8080"),
			Env:           getEnv("APP_ENV", "development"),
			PortalBaseURL: getEnv("PORTAL_BASE_URL", "http://localhost:3000"),
		},
		Fonnte: FonnteConfig{
			APIToken: getEnv("FONNTE_API_TOKEN", ""),
		},
		Database: DatabaseConfig{
			DSN: getEnv("DATABASE_URL", "postgres://pintour:pintour_pass@localhost:5432/pintour_db?sslmode=disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "supersecretkey_change_in_production"),
			ExpirationHours: getEnvInt("JWT_EXPIRATION_HOURS", 8), // PRD §10.3: 8 jam
		},
		Email: EmailConfig{
			ResendAPIKey: getEnv("RESEND_API_KEY", ""),
			FromAddress:  getEnv("MAIL_FROM", "noreply@pintour.app"),
			AppURL:       getEnv("APP_URL", "http://localhost:3000"),
		},
		Supabase: SupabaseConfig{
			URL:        getEnv("SUPABASE_URL", ""),
			ServiceKey: getEnv("SUPABASE_SERVICE_KEY", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
