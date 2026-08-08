package service

import "testing"

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
