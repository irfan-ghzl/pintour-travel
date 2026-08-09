package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestFonnteServiceInit(t *testing.T) {
	svc := NewFonnteService("TESTTOKEN", nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestEmailServiceInit(t *testing.T) {
	svc := NewEmailService("test_key", "noreply@pintour.app")
	if svc == nil {
		t.Fatal("expected non-nil email service")
	}
	if svc.apiKey != "test_key" {
		t.Errorf("expected apiKey=test_key, got %q", svc.apiKey)
	}
}

// TestFonnteSendNoOpWhenUnconfigured verifies that Send() returns nil silently
// when the service has no token or no repo (development mode).
func TestFonnteSendNoOpWhenUnconfigured(t *testing.T) {
	svc := NewFonnteService("", nil)
	err := svc.Send(context.Background(), "628123456789", "Test", "TEST_TYPE", "hello", nil, nil)
	if err != nil {
		t.Errorf("expected nil error when unconfigured, got: %v", err)
	}
}

// TestFonnteNilReceiverDoesNotPanic ensures defensive nil check works.
func TestFonnteNilReceiverDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Send on nil receiver should not panic: %v", r)
		}
	}()
	var svc *FonnteService
	err := svc.Send(context.Background(), "628123456789", "X", "T", "msg", nil, nil)
	if err != nil {
		t.Errorf("expected nil error on nil receiver, got: %v", err)
	}
}

func TestSetRetryDelay(t *testing.T) {
	original := fonnteRetryDelay
	defer func() { fonnteRetryDelay = original }()

	SetRetryDelay(100 * time.Millisecond)
	if fonnteRetryDelay != 100*time.Millisecond {
		t.Errorf("SetRetryDelay didn't apply: got %v", fonnteRetryDelay)
	}
}

// TestFonnteTemplateHelpers verifies all 7 template wrapper methods
// run without panic in development mode.
func TestFonnteTemplateHelpers(t *testing.T) {
	svc := NewFonnteService("", nil)
	ctx := context.Background()

	calls := []struct {
		name string
		fn   func() error
	}{
		{"SendLeadsWelcome", func() error { return svc.SendLeadsWelcome(ctx, "628111", "A", "Pkg", "lid") }},
		{"SendLeadsNotifSales", func() error { return svc.SendLeadsNotifSales(ctx, "628222", "C", "L", "628111", "Pkg", "lid") }},
		{"SendInvoice", func() error { return svc.SendInvoice(ctx, "628111", "A", "INV-1", "1000", "01 Jan", "url", "iid") }},
		{"SendPaymentReminder", func() error { return svc.SendPaymentReminder(ctx, "628111", "A", "INV-1", 1, "iid") }},
		{"SendDocRequest", func() error { return svc.SendDocRequest(ctx, "628111", "A", "Pkg", "url", "pid") }},
		{"SendDocRejected", func() error { return svc.SendDocRejected(ctx, "628111", "A", "passport", "blur", "url", "pid") }},
		{"SendDepartureReminder", func() error {
			return svc.SendDepartureReminder(ctx, "628111", "A", "Pkg", "01 Jan", "H-7", "REMINDER_H7", "pid")
		}},
		{"SendPortalCredentials", func() error { return svc.SendPortalCredentials(ctx, "628111", "A", "rahasia1", "url", "pid") }},
	}

	for _, c := range calls {
		if err := c.fn(); err != nil {
			t.Errorf("%s returned error: %v", c.name, err)
		}
	}
}

// The portal credentials message is the only copy of the password that ever
// leaves the system — portal accounts have no reset flow — so a message that
// renders without it locks the participant out while reporting success. Asserted
// on the rendered text because the gateway's address is not settable yet
// (tiket 08), so nothing sent through Send can be observed.
func TestPortalCredentialsMessageCarriesTheCredential(t *testing.T) {
	const (
		phone    = "628111000001"
		password = "Xk7mQp2r"
		portal   = "https://pintour.test/portal"
	)
	msg := portalCredentialsMessage("Budi", phone, password, portal)

	for _, want := range []string{password, phone, portal, "Budi"} {
		if !strings.Contains(msg, want) {
			t.Errorf("pesan kredensial tidak memuat %q:\n%s", want, msg)
		}
	}
}
