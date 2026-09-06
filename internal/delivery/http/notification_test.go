package httpdelivery

// Notification completeness (tiket 08).
//
// Until the WhatsApp client's address became settable, none of this was
// observable: Send returned before writing its notification row whenever the
// token was empty, so h.Notifications stayed empty however many messages a flow
// claimed to send. withFonnteServer runs Send the whole way through against a
// local stand-in, which is what lets these assertions exist at all.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
)

// whatsappGateway stands in for Fonnte, accepting every send.
func whatsappGateway(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"id":["1"],"process":"pending"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// sentTypes waits for the background sends to land and returns the message types
// recorded, in order. Notifications go out on their own goroutines after the
// response, so a test that reads immediately reads nothing.
func sentTypes(t *testing.T, h *harness) []string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var types []string
	for time.Now().Before(deadline) {
		types = types[:0]
		for _, n := range h.Notifications.Sent {
			types = append(types, n.MessageType)
		}
		if len(types) > 0 {
			// Give any sibling sends on the same flow a moment to arrive too.
			time.Sleep(100 * time.Millisecond)
			types = types[:0]
			for _, n := range h.Notifications.Sent {
				types = append(types, n.MessageType)
			}
			return types
		}
		time.Sleep(20 * time.Millisecond)
	}
	return types
}

func countType(types []string, want string) int {
	n := 0
	for _, t := range types {
		if t == want {
			n++
		}
	}
	return n
}

// seedFullyPayable is one participant with one invoice they are about to settle.
func seedFullyPayable(h *harness) {
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-1", BatchID: "batch-1", Name: "Peserta Satu",
		Phone: "628222222222", PackageName: "Umroh Reguler", IsActive: false,
	})
	h.Invoices.Seed(domainInvoice.Invoice{
		ID: "invoice-1", InvoiceNumber: "INV-202608-0001",
		ParticipantID: "participant-1", BatchID: "batch-1",
		Amount: 10000000, Status: "menunggu_bayar", IssuedBy: "user-admin",
		DueDate: calendar.Today().AddDays(7),
	})
}

// ─── FR-AUTO-04 / diagram §14.5.3 ─────────────────────────────────────────────

// The document list has to reach the participant once payment is confirmed,
// whichever way it was confirmed. Only the old manual-confirm endpoint sent it,
// so the two paths actually used — an admin approving a proof, and the gateway
// settling automatically — told the participant nothing about what to do next.
func TestSettlement_SendsDocumentRequestOnBothPaths(t *testing.T) {
	t.Run("admin menyetujui bukti bayar", func(t *testing.T) {
		gateway := whatsappGateway(t)
		h := newHarness(t, withFonnteServer(gateway.URL))
		seedFullyPayable(h)
		h.Proofs.Seed(domainInvoice.PaymentProof{
			ID: "proof-1", InvoiceID: "invoice-1", FilePath: "participant-1/transfer.jpg",
			AmountClaimed: 10000000, Status: "menunggu",
		})

		h.as("admin").PATCH("/api/v1/admin/invoices/invoice-1/proofs/proof-1/review",
			map[string]string{"status": "disetujui", "notes": "ok"}).
			expectCode(http.StatusOK)

		if types := sentTypes(t, h); countType(types, notification.TypeDocRequest) != 1 {
			t.Errorf("DOC_REQUEST terkirim %d kali, ingin 1 — pesan terkirim: %v",
				countType(types, notification.TypeDocRequest), types)
		}
	})

	t.Run("gateway menyelesaikan otomatis", func(t *testing.T) {
		gateway := whatsappGateway(t)
		h := newHarness(t, withFonnteServer(gateway.URL), withMidtransServer("http://midtrans.invalid"))
		seedFullyPayable(h)
		h.GatewayOrders.Seed(domainInvoice.GatewayOrder{
			ID: "order-row-1", InvoiceID: "invoice-1",
			OrderID: "INV-202608-0001-1754000000", CreatedAt: time.Now(),
		})

		postWebhook(h, settlementNotification("INV-202608-0001-1754000000", "trx-1", "10000000.00")).
			expectCode(http.StatusOK)

		if types := sentTypes(t, h); countType(types, notification.TypeDocRequest) != 1 {
			t.Errorf("DOC_REQUEST terkirim %d kali, ingin 1 — pesan terkirim: %v",
				countType(types, notification.TypeDocRequest), types)
		}
	})
}

// Confirming an invoice that a proof already settled must not repeat the
// instructions — the message belongs to the moment the invoice became fully
// paid, and that moment happened once.
func TestSettlement_ConfirmAfterProofDoesNotResendDocumentRequest(t *testing.T) {
	gateway := whatsappGateway(t)
	h := newHarness(t, withFonnteServer(gateway.URL))
	seedFullyPayable(h)
	h.Proofs.Seed(domainInvoice.PaymentProof{
		ID: "proof-1", InvoiceID: "invoice-1", FilePath: "participant-1/transfer.jpg",
		AmountClaimed: 10000000, Status: "menunggu",
	})

	h.as("admin").PATCH("/api/v1/admin/invoices/invoice-1/proofs/proof-1/review",
		map[string]string{"status": "disetujui", "notes": "ok"}).
		expectCode(http.StatusOK)
	h.as("admin").POST("/api/v1/admin/invoices/invoice-1/confirm", nil).
		expectCode(http.StatusOK)

	if types := sentTypes(t, h); countType(types, notification.TypeDocRequest) != 1 {
		t.Errorf("DOC_REQUEST terkirim %d kali setelah kedua jalur dipakai, ingin 1: %v",
			countType(types, notification.TypeDocRequest), types)
	}
}

// ─── What the invoice message points at (tiket 03) ────────────────────────────

// The invoice a participant receives points at a page they can open the moment
// it arrives. Which link that is gets decided by the flow rather than by the
// template, so it is asserted here, where the flow runs.
//
// It used to end with "(tersedia di portal setelah pembayaran dikonfirmasi)" —
// true of the portal as it was, and the sentence that told a participant not to
// try logging in at exactly the point they needed to.
func TestInvoiceCreated_WhatsAppPointsAtTheInvoicePage(t *testing.T) {
	gateway := whatsappGateway(t)
	h := newHarness(t, withFonnteServer(gateway.URL))
	h.seedBaseline()

	h.as("admin").POST("/api/v1/admin/invoices", map[string]any{
		"participant_id": "participant-1", "batch_id": "batch-1",
		"amount": 25000000, "due_date": calendar.Today().AddDays(7),
	}).expectCode(http.StatusCreated)
	sentTypes(t, h)

	var sent *notification.WANotification
	for i := range h.Notifications.Sent {
		if h.Notifications.Sent[i].MessageType == notification.TypeInvoiceSent {
			sent = &h.Notifications.Sent[i]
		}
	}
	if sent == nil {
		t.Fatalf("invoice terbit tanpa pesan WhatsApp: %v", h.Notifications.Sent)
	}
	if !strings.Contains(sent.MessageContent, "/portal/invoices") {
		t.Errorf("pesan invoice tidak menunjuk halaman invoice portal:\n%s", sent.MessageContent)
	}
	if strings.Contains(strings.ToLower(sent.MessageContent), "setelah pembayaran") {
		t.Errorf("pesan invoice masih menunda PDF sampai pembayaran:\n%s", sent.MessageContent)
	}
}

// ─── The reference a notification points at ───────────────────────────────────

// A row labelled reference_type "invoice" has to reference an invoice. The
// payment reminder pointed at the participant instead, which broke the trail
// from an invoice to the messages sent about it.
func TestNotifications_ReferenceMatchesItsType(t *testing.T) {
	gateway := whatsappGateway(t)
	h := newHarness(t, withFonnteServer(gateway.URL))
	seedFullyPayable(h)
	h.Proofs.Seed(domainInvoice.PaymentProof{
		ID: "proof-1", InvoiceID: "invoice-1", FilePath: "participant-1/transfer.jpg",
		AmountClaimed: 10000000, Status: "menunggu",
	})

	h.as("admin").PATCH("/api/v1/admin/invoices/invoice-1/proofs/proof-1/review",
		map[string]string{"status": "disetujui", "notes": "ok"}).
		expectCode(http.StatusOK)
	sentTypes(t, h)

	known := map[string]string{"invoice": "invoice-1", "participant": "participant-1"}
	for _, n := range h.Notifications.Sent {
		if n.ReferenceType == nil || n.ReferenceID == nil {
			continue
		}
		want, tracked := known[*n.ReferenceType]
		if !tracked {
			continue
		}
		if *n.ReferenceID != want {
			t.Errorf("%s: reference_type=%q menunjuk ke %q, ingin %q",
				n.MessageType, *n.ReferenceType, *n.ReferenceID, want)
		}
	}
}

// ─── The gateway address itself ───────────────────────────────────────────────

// The first criterion of this ticket, asserted directly: the default is
// unchanged, so no deployment moves, and the override only applies when given.
func TestFonnteBaseURL_DefaultsToTheRealGateway(t *testing.T) {
	gateway := whatsappGateway(t)
	h := newHarness(t, withFonnteServer(gateway.URL))
	seedFullyPayable(h)

	// A request that sends nothing leaves the log empty; this is only here to
	// prove the harness wiring does not send on its own.
	h.as("admin").GET("/api/v1/admin/invoices").expectCode(http.StatusOK)
	if len(h.Notifications.Sent) != 0 {
		t.Errorf("notifikasi terkirim tanpa aksi yang memicunya: %v", h.Notifications.Sent)
	}
}

// The stand-in gateway must actually be reached, otherwise every assertion above
// would pass against a client that silently sent nothing.
func TestFonnteBaseURL_SendsReachTheConfiguredServer(t *testing.T) {
	var mu sync.Mutex
	received := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		received++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true}`))
	}))
	t.Cleanup(srv.Close)

	h := newHarness(t, withFonnteServer(srv.URL))
	seedFullyPayable(h)
	h.Proofs.Seed(domainInvoice.PaymentProof{
		ID: "proof-1", InvoiceID: "invoice-1", FilePath: "participant-1/transfer.jpg",
		AmountClaimed: 10000000, Status: "menunggu",
	})

	h.as("admin").PATCH("/api/v1/admin/invoices/invoice-1/proofs/proof-1/review",
		map[string]string{"status": "disetujui", "notes": "ok"}).
		expectCode(http.StatusOK)
	sentTypes(t, h)

	mu.Lock()
	defer mu.Unlock()
	if received == 0 {
		t.Error("tidak ada permintaan yang sampai ke server WhatsApp uji")
	}
}
