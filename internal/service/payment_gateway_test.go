package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The base URL became settable so tests can point the client at a local server.
// The defaults must not have moved: sandbox and production still resolve exactly
// as they did before the option existed.
func TestNewMidtransService_BaseURL(t *testing.T) {
	cases := []struct {
		name string
		env  string
		opts []MidtransOption
		want string
	}{
		{name: "sandbox default", env: "sandbox", want: "https://app.sandbox.midtrans.com"},
		{name: "unset env falls back to sandbox", env: "", want: "https://app.sandbox.midtrans.com"},
		{name: "production default", env: "production", want: "https://app.midtrans.com"},
		{
			name: "override",
			env:  "sandbox",
			opts: []MidtransOption{WithMidtransBaseURL("http://127.0.0.1:8080")},
			want: "http://127.0.0.1:8080",
		},
		{
			name: "override strips trailing slash",
			env:  "sandbox",
			opts: []MidtransOption{WithMidtransBaseURL("http://127.0.0.1:8080/")},
			want: "http://127.0.0.1:8080",
		},
		{
			name: "empty override keeps the default",
			env:  "production",
			opts: []MidtransOption{WithMidtransBaseURL("")},
			want: "https://app.midtrans.com",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewMidtransService("server-key", "client-key", c.env, c.opts...)
			if s.baseURL != c.want {
				t.Errorf("baseURL = %q, want %q", s.baseURL, c.want)
			}
		})
	}
}

// Surel peserta opsional di formulir konsultasi, jadi sebagiannya kosong.
// Mengirimkannya sebagai string kosong membuat Midtrans menolak seluruh
// transaksi dengan "customer_details.email format is invalid", dan peserta hanya
// melihat galat 500 tanpa sebab. Terjadi sungguhan pada peserta tanpa surel.
func TestCreateSnap_OmitsEmailWhenTheParticipantHasNone(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"tok-123"}`))
	}))
	defer srv.Close()

	svc := NewMidtransService("server-key", "client-key", "sandbox", WithMidtransBaseURL(srv.URL))
	if _, err := svc.CreateSnap(context.Background(), SnapRequest{
		OrderID: "INV-1", GrossAmount: 100000, ItemName: "Paket",
		Customer: SnapCustomer{Name: "Budi", Phone: "628111", Email: ""},
	}); err != nil {
		t.Fatalf("CreateSnap: %v", err)
	}

	cust, _ := got["customer_details"].(map[string]any)
	if _, ada := cust["email"]; ada {
		t.Fatalf("email kosong tetap dikirim: %v", cust)
	}
	if cust["first_name"] != "Budi" || cust["phone"] != "628111" {
		t.Fatalf("nama atau telepon hilang: %v", cust)
	}
}

func TestCreateSnap_KeepsEmailWhenPresent(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"tok-123"}`))
	}))
	defer srv.Close()

	svc := NewMidtransService("server-key", "client-key", "sandbox", WithMidtransBaseURL(srv.URL))
	if _, err := svc.CreateSnap(context.Background(), SnapRequest{
		OrderID: "INV-2", GrossAmount: 100000, ItemName: "Paket",
		Customer: SnapCustomer{Name: "Hendro", Phone: "628222", Email: "hendro@example.com"},
	}); err != nil {
		t.Fatalf("CreateSnap: %v", err)
	}

	cust, _ := got["customer_details"].(map[string]any)
	if cust["email"] != "hendro@example.com" {
		t.Fatalf("surel yang ada malah hilang: %v", cust)
	}
}
