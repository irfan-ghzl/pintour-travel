package portaluser

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// §14.4 PortalUser.VerifyPassword(pw) bool
func TestPortalUserVerifyPassword(t *testing.T) {
	h, err := bcrypt.GenerateFromPassword([]byte("Ab3xK9pQ"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	stored := string(h)

	cases := map[string]struct {
		hash     string
		password string
		want     bool
	}{
		"password benar":     {stored, "Ab3xK9pQ", true},
		"password salah":     {stored, "Ab3xK9pR", false},
		"password kosong":    {stored, "", false},
		"hash kosong":        {"", "Ab3xK9pQ", false},
		"hash bukan bcrypt":  {"Ab3xK9pQ", "Ab3xK9pQ", false},
		"beda kapitalisasi":  {stored, "ab3xk9pq", false},
		"password terpotong": {stored, "Ab3xK9p", false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			pu := PortalUser{PasswordHash: tc.hash}
			if got := pu.VerifyPassword(tc.password); got != tc.want {
				t.Errorf("VerifyPassword(%q) = %v, want %v", tc.password, got, tc.want)
			}
		})
	}
}
