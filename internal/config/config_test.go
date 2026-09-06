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
			// Start from a deployment that is otherwise fit for production, so
			// the case under test is the secret and nothing else. Validate now
			// checks the public URLs and the payment gateway too, and a bare
			// Config trips those as well.
			cfg := productionConfig()
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

// ─── Production configuration ─────────────────────────────────────────────────

// Every case here is a deployment that would start cleanly and fail silently:
// tokens anyone can forge, links nobody can open, payments that never arrive.
// Refusing to start is the only signal a misconfigured deployment gets, so what
// is under test is that it refuses.

// productionConfig is a deployment configured correctly, which each case then
// breaks in exactly one way.
func productionConfig() *Config {
	c := &Config{}
	c.Server.Env = EnvProduction
	c.Server.PortalBaseURL = "https://portal.pintour.id"
	c.JWT.Secret = "b8f1c0a94d2e47f6a1b3c5d7e9f0a2b4c6d8e0f2a4b6c8d0e2f4a6b8c0d2e4f6"
	c.Email.AppURL = "https://admin.pintour.id"
	return c
}

func TestValidate_AcceptsAProperProductionConfig(t *testing.T) {
	if err := productionConfig().Validate(); err != nil {
		t.Errorf("a correctly configured production deployment was refused: %v", err)
	}
}

// Development keeps its defaults: localhost URLs and the shared secret are what
// running the project locally means.
func TestValidate_LeavesDevelopmentAlone(t *testing.T) {
	c := &Config{}
	c.Server.Env = "development"
	c.Server.PortalBaseURL = "http://localhost:3000"
	c.JWT.Secret = defaultJWTSecret
	c.Midtrans.ServerKey = "SB-Mid-server-xxx"
	c.Midtrans.Env = "sandbox"

	if err := c.Validate(); err != nil {
		t.Errorf("development was refused: %v", err)
	}
}

func TestValidate_RefusesUnsafeProductionValues(t *testing.T) {
	cases := []struct {
		name    string
		break_  func(*Config)
		mustSay string
	}{
		{
			name:    "secret bawaan",
			break_:  func(c *Config) { c.JWT.Secret = defaultJWTSecret },
			mustSay: "JWT_SECRET",
		},
		{
			name:    "secret kosong",
			break_:  func(c *Config) { c.JWT.Secret = "" },
			mustSay: "JWT_SECRET",
		},
		{
			name:    "portal menunjuk localhost",
			break_:  func(c *Config) { c.Server.PortalBaseURL = "http://localhost:3000" },
			mustSay: "PORTAL_BASE_URL",
		},
		{
			name:    "portal menunjuk 127.0.0.1",
			break_:  func(c *Config) { c.Server.PortalBaseURL = "http://127.0.0.1:3000" },
			mustSay: "PORTAL_BASE_URL",
		},
		{
			name:    "app url kosong",
			break_:  func(c *Config) { c.Email.AppURL = "" },
			mustSay: "APP_URL",
		},
		{
			name:    "app url bukan URL absolut",
			break_:  func(c *Config) { c.Email.AppURL = "admin.pintour.id" },
			mustSay: "APP_URL",
		},
		{
			name: "gateway sungguhan diarahkan ke sandbox",
			break_: func(c *Config) {
				c.Midtrans.ServerKey = "Mid-server-real"
				c.Midtrans.Env = "sandbox"
			},
			mustSay: "MIDTRANS_ENV",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := productionConfig()
			tc.break_(c)

			err := c.Validate()
			if err == nil {
				t.Fatal("production started with a configuration that fails silently")
			}
			if !strings.Contains(err.Error(), tc.mustSay) {
				t.Errorf("error = %q, want it to name %s so the operator knows what to change",
					err, tc.mustSay)
			}
		})
	}
}

// A deployment that does not take online payments says so by leaving the key
// empty; the sandbox setting is then irrelevant and must not block a start.
func TestValidate_AllowsProductionWithoutAPaymentGateway(t *testing.T) {
	c := productionConfig()
	c.Midtrans.ServerKey = ""
	c.Midtrans.Env = "sandbox"

	if err := c.Validate(); err != nil {
		t.Errorf("production without a payment gateway was refused: %v", err)
	}
}

// A demonstration runs the real flow against the sandbox on purpose — a
// defence, a walkthrough for a client — and saying so is what separates it from
// the deployment this check exists to stop.
func TestValidate_AllowsTheSandboxWhenTheDeploymentAsksForIt(t *testing.T) {
	c := productionConfig()
	c.Midtrans.ServerKey = "SB-Mid-server-xxx"
	c.Midtrans.Env = "sandbox"
	c.Midtrans.AllowSandbox = true

	if err := c.Validate(); err != nil {
		t.Errorf("a deployment that opted into the sandbox was refused: %v", err)
	}
}

// The waiver is the whole reason this is opt-in rather than a looser rule, so
// it has to stay narrow: it excuses the sandbox and nothing else.
func TestValidate_SandboxWaiverExcusesNothingElse(t *testing.T) {
	c := productionConfig()
	c.Midtrans.ServerKey = "SB-Mid-server-xxx"
	c.Midtrans.Env = "sandbox"
	c.Midtrans.AllowSandbox = true
	c.JWT.Secret = defaultJWTSecret
	c.Server.PortalBaseURL = "http://localhost:3000"

	err := c.Validate()
	if err == nil {
		t.Fatal("the sandbox waiver let an unsafe production configuration start")
	}
	for _, want := range []string{"JWT_SECRET", "PORTAL_BASE_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to still name %s", err, want)
		}
	}
	if strings.Contains(err.Error(), "MIDTRANS_ENV") {
		t.Errorf("error = %q, want the sandbox itself to be excused", err)
	}
}

// Not set means not waived. A deployment reaches the sandbox only by asking
// for it in those exact words, never by a value that merely looks affirmative.
func TestMidtransAllowSandbox_IsOffUnlessSaidExactly(t *testing.T) {
	for _, raw := range []string{"", "false", "1", "yes", "TRUE", "True"} {
		t.Run("MIDTRANS_ALLOW_SANDBOX="+raw, func(t *testing.T) {
			t.Setenv("MIDTRANS_ALLOW_SANDBOX", raw)

			if Load().Midtrans.AllowSandbox {
				t.Errorf("%q waived the sandbox check; only \"true\" may", raw)
			}
		})
	}

	t.Run("MIDTRANS_ALLOW_SANDBOX=true", func(t *testing.T) {
		t.Setenv("MIDTRANS_ALLOW_SANDBOX", "true")

		if !Load().Midtrans.AllowSandbox {
			t.Error(`"true" did not waive the sandbox check`)
		}
	})
}

// Several mistakes at once are reported together. An operator fixing a
// misconfigured deployment should not have to restart it once per problem.
func TestValidate_ReportsEveryProblemAtOnce(t *testing.T) {
	c := productionConfig()
	c.JWT.Secret = defaultJWTSecret
	c.Server.PortalBaseURL = "http://localhost:3000"
	c.Email.AppURL = ""

	err := c.Validate()
	if err == nil {
		t.Fatal("expected a refusal")
	}
	for _, want := range []string{"JWT_SECRET", "PORTAL_BASE_URL", "APP_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
}
