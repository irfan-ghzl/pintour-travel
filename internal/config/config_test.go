package config

import (
	"strings"
	"testing"
)

// An environment variable set to empty is not a value. Three readers used to
// disagree about that, and the one that returned the empty string produced
// links that began with "/portal" — resolving against whatever host opened
// them.
func TestEnv_EmptyCountsAsUnset(t *testing.T) {
	cases := map[string]struct {
		set      bool
		value    string
		fallback string
		want     string
	}{
		"tidak di-set":      {false, "", "bawaan", "bawaan"},
		"di-set berisi":     {true, "nilai", "bawaan", "nilai"},
		"di-set kosong":     {true, "", "bawaan", "bawaan"},
		"di-set spasi":      {true, " ", "bawaan", " "},
		"bawaan pun kosong": {false, "", "", ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			const key = "PINTOUR_TEST_ENV"
			if tc.set {
				t.Setenv(key, tc.value)
			}
			if got := Env(key, tc.fallback); got != tc.want {
				t.Errorf("Env = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEnvInt(t *testing.T) {
	const key = "PINTOUR_TEST_INT"
	cases := map[string]struct {
		value string
		want  int
	}{
		"angka":          {"42", 42},
		"nol":            {"0", 0},
		"negatif":        {"-3", -3},
		"bukan angka":    {"banyak", 7},
		"kosong":         {"", 7},
		"angka berspasi": {" 42", 7},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(key, tc.value)
			if got := EnvInt(key, 7); got != tc.want {
				t.Errorf("EnvInt(%q) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

func TestEnvFloat(t *testing.T) {
	const key = "PINTOUR_TEST_FLOAT"
	cases := map[string]struct {
		value string
		want  float64
	}{
		"desimal":     {"0.9", 0.9},
		"bulat":       {"1", 1},
		"bukan angka": {"tinggi", 0.85},
		"kosong":      {"", 0.85},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(key, tc.value)
			if got := EnvFloat(key, 0.85); got != tc.want {
				t.Errorf("EnvFloat(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

// The two base URLs appear in links inside WhatsApp messages and emails, and
// used to be read at five call sites that each carried their own default — so a
// deployment that set neither sent participants to two different hosts.
func TestBaseURLs_HaveOneDefaultEach(t *testing.T) {
	if got := PortalBaseURL(); got != "http://localhost:3000" {
		t.Errorf("PortalBaseURL() = %q", got)
	}
	if got := AppURL(); got != "http://localhost:5173" {
		t.Errorf("AppURL() = %q", got)
	}

	t.Setenv("PORTAL_BASE_URL", "https://portal.pintour.test")
	t.Setenv("APP_URL", "https://app.pintour.test")
	if got := PortalBaseURL(); got != "https://portal.pintour.test" {
		t.Errorf("PortalBaseURL() = %q setelah di-set", got)
	}
	if got := AppURL(); got != "https://app.pintour.test" {
		t.Errorf("AppURL() = %q setelah di-set", got)
	}
}

// Load reads every section, and a deployment that sets nothing still starts —
// with development defaults, which Validate is what refuses in production.
func TestLoad_FillsEverySectionWithDefaults(t *testing.T) {
	cfg := Load()

	if cfg.Server.Port == "" {
		t.Error("Server.Port kosong")
	}
	if cfg.JWT.Secret == "" {
		t.Error("JWT.Secret kosong")
	}
	if cfg.JWT.ExpirationHours <= 0 {
		t.Errorf("JWT.ExpirationHours = %d, want positif", cfg.JWT.ExpirationHours)
	}
	if cfg.OCR.ConfidenceThreshold <= 0 || cfg.OCR.ConfidenceThreshold > 1 {
		t.Errorf("OCR.ConfidenceThreshold = %v, want di antara 0 dan 1", cfg.OCR.ConfidenceThreshold)
	}
	if cfg.Database.DSN == "" {
		t.Error("DSN basis data kosong")
	}
}

func TestLoad_ReadsTheEnvironment(t *testing.T) {
	t.Setenv("SERVER_PORT", "9999")
	t.Setenv("JWT_SECRET", "rahasia-yang-panjang-dan-acak")
	t.Setenv("JWT_EXPIRATION_HOURS", "48")
	t.Setenv("OCR_CONFIDENCE_THRESHOLD", "0.95")

	cfg := Load()
	if cfg.Server.Port != "9999" {
		t.Errorf("Port = %q, want 9999", cfg.Server.Port)
	}
	if cfg.JWT.ExpirationHours != 48 {
		t.Errorf("ExpirationHours = %d, want 48", cfg.JWT.ExpirationHours)
	}
	if cfg.OCR.ConfidenceThreshold != 0.95 {
		t.Errorf("ConfidenceThreshold = %v, want 0.95", cfg.OCR.ConfidenceThreshold)
	}
}

// A production deployment refuses to start on the development JWT secret.
// Anyone holding it can forge a super_admin token.
func TestValidate_RefusesTheDevelopmentSecretInProduction(t *testing.T) {
	cases := map[string]struct {
		env     string
		secret  string
		wantErr bool
	}{
		"produksi dengan rahasia bawaan":  {"production", defaultJWTSecret, true},
		"produksi tanpa rahasia":          {"production", "", true},
		"produksi dengan rahasia sendiri": {"production", "rahasia-acak-yang-panjang", false},
		"pengembangan dengan bawaan":      {"development", defaultJWTSecret, false},
		"pengembangan tanpa rahasia":      {"development", "", false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := &Config{}
			cfg.Server.Env = tc.env
			cfg.JWT.Secret = tc.secret

			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("Validate() = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "JWT_SECRET") {
				t.Errorf("pesan %q tidak menyebut JWT_SECRET", err)
			}
		})
	}
}
