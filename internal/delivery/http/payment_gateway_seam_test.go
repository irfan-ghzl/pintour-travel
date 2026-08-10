package httpdelivery

// The one production change ticket 01 allows is a settable payment-gateway base
// URL. These tests prove what it buys: with the gateway pointed at a local test
// server the whole "Bayar Online" path — portal token, middleware, handler,
// invoice service, outbound Snap call — runs inside `go test`, and with the
// gateway unconfigured (the harness default) nothing leaves the process.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// snapStub stands in for the Midtrans Snap API and records what it was sent.
type snapStub struct {
	*httptest.Server
	requests []map[string]any
}

func newSnapStub(t *testing.T, token string) *snapStub {
	t.Helper()
	stub := &snapStub{}
	stub.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("snap stub: decode request: %v", err)
		}
		stub.requests = append(stub.requests, payload)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
	}))
	t.Cleanup(stub.Close)
	return stub
}

func TestPortalCreatePayment_ReachesTestGateway(t *testing.T) {
	stub := newSnapStub(t, "snap-token-abc")
	h := newHarness(t, withMidtransServer(stub.URL))
	h.seedBaseline()

	res := h.asParticipant("participant-1").
		POST("/api/v1/portal/invoices/invoice-1/create-payment", nil).
		expectCode(http.StatusOK)

	if got := res.data()["snap_token"]; got != "snap-token-abc" {
		t.Errorf("snap_token = %v, want snap-token-abc", got)
	}
	if len(stub.requests) != 1 {
		t.Fatalf("gateway received %d transactions, want 1", len(stub.requests))
	}

	// The stored order id is what a later webhook is matched against, so the
	// token and order id must both have been written back to the invoice.
	inv, err := h.Invoices.GetByID(t.Context(), "invoice-1")
	if err != nil {
		t.Fatalf("read back invoice: %v", err)
	}
	if inv.SnapToken != "snap-token-abc" || inv.MidtransOrderID == "" {
		t.Errorf("invoice snap fields = (%q, %q), want token stored with a non-empty order id",
			inv.SnapToken, inv.MidtransOrderID)
	}
}

// Another participant's invoice is reported as missing rather than forbidden, so
// the existence of other participants' invoices is not revealed.
func TestPortalCreatePayment_RefusesAnotherParticipantsInvoice(t *testing.T) {
	stub := newSnapStub(t, "snap-token-abc")
	h := newHarness(t, withMidtransServer(stub.URL))
	h.seedBaseline()

	h.asParticipant("participant-2").
		POST("/api/v1/portal/invoices/invoice-1/create-payment", nil).
		expectCode(http.StatusNotFound)

	if len(stub.requests) != 0 {
		t.Errorf("gateway received %d transactions, want none", len(stub.requests))
	}
}

// Default harness: no gateway keys, so the client refuses before any outbound
// call — the same behaviour as a deployment with MIDTRANS_SERVER_KEY unset.
func TestPortalCreatePayment_WithoutGatewayConfigured(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	h.asParticipant("participant-1").
		POST("/api/v1/portal/invoices/invoice-1/create-payment", nil).
		expectCode(http.StatusInternalServerError)
}
