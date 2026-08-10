package user

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func hashOf(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash %q: %v", plain, err)
	}
	return string(h)
}

// §14.4 User.Authenticate(password) bool
func TestUserAuthenticate(t *testing.T) {
	stored := hashOf(t, "rahasia123")

	cases := map[string]struct {
		hash     string
		password string
		want     bool
	}{
		"password benar":          {stored, "rahasia123", true},
		"password salah":          {stored, "rahasia124", false},
		"password kosong":         {stored, "", false},
		"hash kosong":             {"", "rahasia123", false},
		"hash bukan bcrypt":       {"plainteks", "plainteks", false},
		"keduanya kosong":         {"", "", false},
		"beda kapitalisasi":       {stored, "Rahasia123", false},
		"password dengan spasi":   {stored, " rahasia123", false},
		"hash dengan spasi depan": {" " + stored, "rahasia123", false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			u := User{Password: tc.hash}
			if got := u.Authenticate(tc.password); got != tc.want {
				t.Errorf("Authenticate(%q) = %v, want %v", tc.password, got, tc.want)
			}
		})
	}
}

// §14.4 User.HasRole(role) bool
func TestUserHasRole(t *testing.T) {
	cases := map[string]struct {
		role    string
		allowed []string
		want    bool
	}{
		"peran cocok satu-satunya":   {"admin", []string{"admin"}, true},
		"peran cocok di daftar":      {"konsultan", []string{"super_admin", "admin", "konsultan"}, true},
		"peran tidak di daftar":      {"tour_leader", []string{"super_admin", "admin"}, false},
		"daftar kosong menolak":      {"super_admin", nil, false},
		"peran kosong ditolak":       {"", []string{"admin"}, false},
		"peran kosong vs daftar nil": {"", nil, false},
		"beda kapitalisasi ditolak":  {"Admin", []string{"admin"}, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			u := User{Role: tc.role}
			if got := u.HasRole(tc.allowed...); got != tc.want {
				t.Errorf("HasRole(%v) with role %q = %v, want %v", tc.allowed, tc.role, got, tc.want)
			}
		})
	}
}
