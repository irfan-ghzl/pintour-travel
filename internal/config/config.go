package config

import (
	"fmt"
	"os"
	"strconv"
)

// defaultJWTSecret is the insecure development fallback. It must never be used
// in production — Validate() enforces this.
const defaultJWTSecret = "supersecretkey_change_in_production"

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Fonnte   FonnteConfig
	Email    EmailConfig
	Supabase SupabaseConfig
	Midtrans MidtransConfig
	Chatbot  ChatbotConfig
	OCR      OCRConfig
}

// MidtransConfig — payment gateway (v2.0 F1).
type MidtransConfig struct {
	ServerKey string
	ClientKey string
	Env       string // "sandbox" | "production"
}

// ChatbotConfig — Gemini-powered WA chatbot (v2.0 F2).
type ChatbotConfig struct {
	GeminiKey    string
	Model        string
	MaxHistory   int
	Active       bool
	WebhookToken string // shared secret to authenticate Fonnte inbound webhook
}

// OCRConfig — document OCR (v2.0 F6). Default engine is the self-hosted
// Tesseract sidecar so no document image is sent to a third party.
type OCRConfig struct {
	Engine              string // "tesseract_local" | "google_vision" | "none"
	GoogleVisionKey     string
	TesseractURL        string
	TempDir             string
	ConfidenceThreshold float64
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
			Port:          Env("SERVER_PORT", "8080"),
			Env:           Env("APP_ENV", "development"),
			PortalBaseURL: Env("PORTAL_BASE_URL", "http://localhost:3000"),
		},
		Fonnte: FonnteConfig{
			APIToken: Env("FONNTE_API_TOKEN", ""),
		},
		Database: DatabaseConfig{
			DSN: Env("DATABASE_URL", "postgres://pintour:pintour_pass@localhost:5432/pintour_db?sslmode=disable"),
		},
		Redis: RedisConfig{
			Addr:     Env("REDIS_ADDR", "localhost:6379"),
			Password: Env("REDIS_PASSWORD", ""),
			DB:       EnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          Env("JWT_SECRET", defaultJWTSecret),
			ExpirationHours: EnvInt("JWT_EXPIRATION_HOURS", 8), // PRD §10.3: 8 jam
		},
		Email: EmailConfig{
			ResendAPIKey: Env("RESEND_API_KEY", ""),
			FromAddress:  Env("MAIL_FROM", "noreply@pintour.app"),
			AppURL:       Env("APP_URL", "http://localhost:3000"),
		},
		Supabase: SupabaseConfig{
			URL:        Env("SUPABASE_URL", ""),
			ServiceKey: Env("SUPABASE_SERVICE_KEY", ""),
		},
		Midtrans: MidtransConfig{
			ServerKey: Env("MIDTRANS_SERVER_KEY", ""),
			ClientKey: Env("MIDTRANS_CLIENT_KEY", ""),
			Env:       Env("MIDTRANS_ENV", "sandbox"),
		},
		Chatbot: ChatbotConfig{
			GeminiKey:    Env("GEMINI_API_KEY", ""),
			Model:        Env("CHATBOT_MODEL", "gemini-1.5-flash"),
			MaxHistory:   EnvInt("CHATBOT_MAX_HISTORY", 10),
			Active:       Env("CHATBOT_ACTIVE", "true") == "true",
			WebhookToken: Env("FONNTE_WEBHOOK_TOKEN", ""),
		},
		// OCR config (v2.0 F6) — default self-hosted Tesseract sidecar.
		OCR: OCRConfig{
			Engine:              Env("OCR_ENGINE", "tesseract_local"),
			GoogleVisionKey:     Env("GOOGLE_VISION_API_KEY", ""),
			TesseractURL:        Env("TESSERACT_API_URL", "http://tesseract-api:8884"),
			TempDir:             Env("OCR_TEMP_DIR", os.TempDir()),
			ConfidenceThreshold: EnvFloat("OCR_CONFIDENCE_THRESHOLD", 0.85),
		},
	}
}

// Validate checks for unsafe configuration. In production it refuses to run
// with a missing or default JWT secret, which would allow token forgery.
func (c *Config) Validate() error {
	if c.Server.Env == "production" {
		if c.JWT.Secret == "" || c.JWT.Secret == defaultJWTSecret {
			return fmt.Errorf("JWT_SECRET must be set to a strong random value in production")
		}
	}
	return nil
}

// Env returns the environment variable key, or fallback when it is unset or
// empty.
//
// Empty counts as unset on purpose. An empty PORTAL_BASE_URL is not a base URL,
// and treating it as one produced links that started with "/portal" — which
// resolve against whatever host happened to open them. Three separate readers
// existed before this one, and they disagreed on exactly this point.
func Env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// EnvInt is Env for a whole number, falling back when the value is missing or
// not a number.
func EnvInt(key string, fallback int) int {
	if v, err := strconv.Atoi(Env(key, "")); err == nil {
		return v
	}
	return fallback
}

// EnvFloat is Env for a decimal.
func EnvFloat(key string, fallback float64) float64 {
	if v, err := strconv.ParseFloat(Env(key, ""), 64); err == nil {
		return v
	}
	return fallback
}

// PortalBaseURL is where the participant portal is served, and AppURL where the
// admin app is. Both appear in links inside WhatsApp messages and emails, and
// both used to be read at five call sites that each carried their own default —
// so a deployment that set neither sent participants to two different hosts.
func PortalBaseURL() string { return Env("PORTAL_BASE_URL", "http://localhost:3000") }

// AppURL is the admin application's base URL.
func AppURL() string { return Env("APP_URL", "http://localhost:5173") }
