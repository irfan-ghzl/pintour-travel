package httpdelivery

import (
	"net/http"
	"testing"

	"golang.org/x/crypto/bcrypt"

	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

// Mereset kata sandi dulu menjawab "Password berhasil direset" tanpa mengubah
// apa pun: handler menetapkan hash baru pada entitas lalu memanggil Update, dan
// Update hanya menulis nama, surel, peran, dan telepon. Kata sandi lama tetap
// berlaku, dan tidak ada satu pun tanda bahwa resetnya gagal.
//
// Diuji lewat perilaku yang teramati — kata sandi lama harus berhenti bekerja
// dan yang baru harus bekerja — bukan dengan memeriksa kolomnya langsung.

func seedStaff(h *harness, id, email, password, role string) {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	h.Users.Seed(domainUser.User{
		ID: id, Name: "Staf Uji", Email: email, Password: string(hashed),
		Role: role, IsActive: true,
	})
}

func login(h *harness, email, password string) *response {
	return h.anonymous().POST("/api/v1/auth/login", map[string]string{
		"email": email, "password": password,
	})
}

func TestResetPasswordAdmin_ActuallyReplacesThePassword(t *testing.T) {
	h := newHarness(t)
	seedStaff(h, "staf-1", "budi@pintour.com", "lamaSekali1", "konsultan")

	h.as("super_admin").
		PATCH("/api/v1/admin/users/staf-1/reset-password", map[string]string{"password": "rahasia123"}).
		expectCode(http.StatusOK)

	login(h, "budi@pintour.com", "rahasia123").expectCode(http.StatusOK)
	login(h, "budi@pintour.com", "lamaSekali1").expectCode(http.StatusUnauthorized)
}

func TestResetPasswordAdmin_RefusesAnyoneBelowSuperAdmin(t *testing.T) {
	for _, role := range []string{"admin", "konsultan", "tour_leader"} {
		h := newHarness(t)
		seedStaff(h, "staf-1", "budi@pintour.com", "lamaSekali1", "konsultan")

		h.as(role).
			PATCH("/api/v1/admin/users/staf-1/reset-password", map[string]string{"password": "rahasia123"}).
			expectCode(http.StatusForbidden)

		// Kata sandi lama harus tetap berlaku setelah percobaan yang ditolak.
		login(h, "budi@pintour.com", "lamaSekali1").expectCode(http.StatusOK)
	}
}
