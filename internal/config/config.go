package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// EnvProduction is the APP_ENV value that turns on the production posture:
// Secure cookies, and the configuration checks in Validate.
const EnvProduction = "production"

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

	// AllowSandbox lets a production deployment run against the payment
	// sandbox on purpose. It exists for demonstrations — a defence, a
	// walkthrough for a client — where the whole point is to show the payment
	// flow without moving money. Off unless set, so no deployment reaches this
	// state by drifting into it.
	AllowSandbox bool
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
			ServerKey:    Env("MIDTRANS_SERVER_KEY", ""),
			ClientKey:    Env("MIDTRANS_CLIENT_KEY", ""),
			Env:          Env("MIDTRANS_ENV", "sandbox"),
			AllowSandbox: Env("MIDTRANS_ALLOW_SANDBOX", "") == "true",
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
	if c.Server.Env != EnvProduction {
		return nil
	}

	// Every check below guards a failure that produces no error at runtime. A
	// deployment misconfigured in any of these ways starts cleanly, serves
	// requests, and logs nothing — while sending participants links that go
	// nowhere or taking payments that never arrive. Refusing to start is the
	// only signal left.
	var problems []string

	if c.JWT.Secret == "" || c.JWT.Secret == defaultJWTSecret {
		problems = append(problems,
			"JWT_SECRET masih kosong atau memakai nilai bawaan — token bisa dipalsukan siapa pun yang tahu nilai itu")
	}

	// These two are pasted into WhatsApp messages and password-reset emails. A
	// localhost address in production is a link every recipient cannot open, and
	// nothing about sending it fails.
	for _, u := range []struct{ key, value string }{
		{"PORTAL_BASE_URL", c.Server.PortalBaseURL},
		{"APP_URL", c.Email.AppURL},
	} {
		if problem := checkPublicURL(u.key, u.value); problem != "" {
			problems = append(problems, problem)
		}
	}

	// A production deployment pointed at the payment sandbox accepts payments
	// that never reach a bank account, and reports every one of them as settled.
	// Leaving MIDTRANS_SERVER_KEY empty is a legitimate deployment — one that
	// does not take online payments, and says so by failing the endpoint — but
	// a real key against the sandbox is not.
	//
	// MIDTRANS_ALLOW_SANDBOX=true waives this, and only this. A deployment that
	// says so has declared it is a demonstration: the sandbox is the intent,
	// not an oversight. Everything the check protects against still holds for
	// anyone who does not set it, which is the point of making it opt-in rather
	// than loosening the rule for all.
	if c.Midtrans.ServerKey != "" && c.Midtrans.Env != EnvProduction && !c.Midtrans.AllowSandbox {
		problems = append(problems, fmt.Sprintf(
			"MIDTRANS_ENV=%q padahal MIDTRANS_SERVER_KEY terisi — pembayaran akan diproses di sandbox "+
				"dan dilaporkan lunas tanpa uang masuk; set MIDTRANS_ENV=production, atau kosongkan "+
				"server key bila deployment ini memang tidak menerima pembayaran online",
			c.Midtrans.Env))
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("konfigurasi produksi tidak aman:\n  - %s", strings.Join(problems, "\n  - "))
}

// checkPublicURL reports why a URL cannot be handed to someone outside the
// deployment, or "" when it can.
func checkPublicURL(key, value string) string {
	if value == "" {
		return key + " kosong — tautan yang dikirim ke peserta akan menunjuk ke mana-mana"
	}
	for _, local := range []string{"localhost", "127.0.0.1", "0.0.0.0", "::1"} {
		if strings.Contains(value, local) {
			return fmt.Sprintf("%s=%q menunjuk ke mesin lokal — peserta yang menerimanya tidak bisa membukanya", key, value)
		}
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return fmt.Sprintf("%s=%q bukan URL absolut", key, value)
	}
	return ""
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
