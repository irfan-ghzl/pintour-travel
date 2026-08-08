package httpdelivery

// Payment gateway accuracy (tiket 07).
//
// The webhook is the one endpoint where being wrong costs money rather than
// causing an error. Nothing here fails loudly: the double-counting defect made
// invoices settle themselves, and the overwritten-order defect made payments
// vanish — both while every screen looked normal.

import (
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
)

// The harness's Midtrans server key, as withMidtransServer sets it. The webhook
// verifies a SHA-512 signature over order id + status code + amount + this key,
// so a test has to sign its notifications the way Midtrans does.
const testMidtransServerKey = "SB-Mid-server-test"

// midtransSignature reproduces the gateway's signature so the webhook accepts
// the notification and the test exercises the path behind it, not the guard.
func midtransSignature(orderID, statusCode, grossAmount string) string {
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + testMidtransServerKey))
	return hex.EncodeToString(sum[:])
}

// settlementNotification is one Midtrans settlement callback.
func settlementNotification(orderID, transactionID, grossAmount string) map[string]string {
	return map[string]string{
		"order_id":           orderID,
		"transaction_id":     transactionID,
		"transaction_status": "settlement",
		"fraud_status":       "accept",
		"payment_type":       "bank_transfer",
		"gross_amount":       grossAmount,
		"status_code":        "200",
		"signature_key":      midtransSignature(orderID, "200", grossAmount),
	}
}

// seedPayable puts an unpaid 10,000,000 invoice in front of an open Snap
// session, which is the state a gateway notification arrives into.
func seedPayable(h *harness) {
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-1", BatchID: "batch-1", Name: "Peserta Satu",
		Phone: "628222222222", PackageName: "Umroh Reguler", IsActive: false,
	})
	h.Invoices.Seed(domainInvoice.Invoice{
		ID: "invoice-1", InvoiceNumber: "INV-202608-0001",
		ParticipantID: "participant-1", BatchID: "batch-1",
		Amount: 10000000, Status: "menunggu_bayar", IssuedBy: "user-admin",
		DueDate: time.Now().Add(7 * 24 * time.Hour),
	})
	h.GatewayOrders.Seed(domainInvoice.GatewayOrder{
		ID: "order-row-1", InvoiceID: "invoice-1",
		OrderID: "INV-202608-0001-1754000000", CreatedAt: time.Now(),
	})
}

func postWebhook(h *harness, body map[string]string) *response {
	return h.anonymous().POST("/api/v1/webhooks/midtrans", body)
}

// ─── Double counting ──────────────────────────────────────────────────────────

// The defect this ticket exists for. Processing stopped early only when the
// invoice was already "lunas"; a partly-paid invoice is still "menunggu_bayar",
// so every redelivery of the same settlement wrote another proof for the same
// money — and four retries of one down payment settled a ten-million invoice.
func TestGatewayWebhook_RepeatedSettlementCountsOnce(t *testing.T) {
	h := newHarness(t, withMidtransServer("http://midtrans.invalid"))
	seedPayable(h)

	// One 2,500,000 down payment, delivered five times.
	notification := settlementNotification("INV-202608-0001-1754000000", "trx-1", "2500000.00")
	for i := 0; i < 5; i++ {
		postWebhook(h, notification).expectCode(http.StatusOK)
	}

	proofs, err := h.Proofs.GetByInvoice(t.Context(), "invoice-1")
	if err != nil {
		t.Fatalf("baca bukti bayar: %v", err)
	}
	if len(proofs) != 1 {
		t.Errorf("bukti bayar = %d, ingin 1 — uang muka yang sama dihitung ulang", len(proofs))
	}

	inv, err := h.Invoices.GetByID(t.Context(), "invoice-1")
	if err != nil {
		t.Fatalf("baca invoice: %v", err)
	}
	if inv.Status == "lunas" {
		t.Error("invoice dinyatakan lunas padahal baru dibayar 2,5jt dari 10jt")
	}
	if h.Participants.participants["participant-1"].IsActive {
		t.Error("portal peserta diaktifkan atas pembayaran sebagian")
	}
}

// Two genuinely different transactions against one invoice are two payments,
// not a redelivery — the idempotency key is the transaction, not the invoice.
func TestGatewayWebhook_DistinctTransactionsBothCount(t *testing.T) {
	h := newHarness(t, withMidtransServer("http://midtrans.invalid"))
	seedPayable(h)

	postWebhook(h, settlementNotification("INV-202608-0001-1754000000", "trx-1", "4000000.00")).
		expectCode(http.StatusOK)
	postWebhook(h, settlementNotification("INV-202608-0001-1754000000", "trx-2", "6000000.00")).
		expectCode(http.StatusOK)

	proofs, _ := h.Proofs.GetByInvoice(t.Context(), "invoice-1")
	if len(proofs) != 2 {
		t.Fatalf("bukti bayar = %d, ingin 2", len(proofs))
	}
	inv, _ := h.Invoices.GetByID(t.Context(), "invoice-1")
	if inv.Status != "lunas" {
		t.Errorf("status invoice = %q, ingin lunas setelah 4jt + 6jt", inv.Status)
	}
}

// ─── The overwritten order ────────────────────────────────────────────────────

// A participant who opens the payment page, reloads, then pays in the first tab
// sends the FIRST session's order id. While an invoice remembered only its
// latest order, that notification matched nothing: the gateway took the money
// and the system never recorded it.
func TestGatewayWebhook_NotificationForEarlierSessionStillMatches(t *testing.T) {
	h := newHarness(t, withMidtransServer("http://midtrans.invalid"))
	seedPayable(h)

	// A second session opened later, as a reload would.
	h.GatewayOrders.Seed(domainInvoice.GatewayOrder{
		ID: "order-row-2", InvoiceID: "invoice-1",
		OrderID: "INV-202608-0001-1754009999", CreatedAt: time.Now(),
	})

	// Payment completed in the FIRST tab.
	postWebhook(h, settlementNotification("INV-202608-0001-1754000000", "trx-lama", "10000000.00")).
		expectCode(http.StatusOK)

	proofs, _ := h.Proofs.GetByInvoice(t.Context(), "invoice-1")
	if len(proofs) != 1 {
		t.Fatalf("bukti bayar = %d, ingin 1 — pembayaran sesi lama hilang", len(proofs))
	}
	inv, _ := h.Invoices.GetByID(t.Context(), "invoice-1")
	if inv.Status != "lunas" {
		t.Errorf("status invoice = %q, ingin lunas", inv.Status)
	}
}

// ─── Fraud review ─────────────────────────────────────────────────────────────

// The guard used to exempt bank transfers, which let through exactly the
// transactions a reviewer had flagged.
func TestGatewayWebhook_FraudFlaggedTransactionIsHeld(t *testing.T) {
	for _, paymentType := range []string{"bank_transfer", "credit_card", "gopay"} {
		t.Run(paymentType, func(t *testing.T) {
			h := newHarness(t, withMidtransServer("http://midtrans.invalid"))
			seedPayable(h)

			body := settlementNotification("INV-202608-0001-1754000000", "trx-1", "10000000.00")
			body["fraud_status"] = "challenge"
			body["payment_type"] = paymentType
			postWebhook(h, body).expectCode(http.StatusOK)

			proofs, _ := h.Proofs.GetByInvoice(t.Context(), "invoice-1")
			if len(proofs) != 0 {
				t.Errorf("bukti bayar dibuat (%d) atas transaksi yang ditahan review fraud", len(proofs))
			}
			inv, _ := h.Invoices.GetByID(t.Context(), "invoice-1")
			if inv.Status == "lunas" {
				t.Error("invoice lunas atas transaksi yang ditahan review fraud")
			}
		})
	}
}

// ─── Database failure vs unknown order ────────────────────────────────────────

// An unknown order is final; a broken database is not. Answering 200 to the
// second tells the gateway the payment was delivered and stops the retries,
// which is the one reply that loses it for good.
func TestGatewayWebhook_DatabaseFailureAsksForRetry(t *testing.T) {
	h := newHarness(t, withMidtransServer("http://midtrans.invalid"))
	seedPayable(h)
	h.GatewayOrders.Fail(errWriteFailed)

	postWebhook(h, settlementNotification("INV-202608-0001-1754000000", "trx-1", "2500000.00")).
		expectCode(http.StatusInternalServerError)
}

func TestGatewayWebhook_UnknownOrderIsNotFound(t *testing.T) {
	h := newHarness(t, withMidtransServer("http://midtrans.invalid"))
	seedPayable(h)

	postWebhook(h, settlementNotification("ORDER-TIDAK-DIKENAL", "trx-1", "2500000.00")).
		expectCode(http.StatusNotFound)
}

// ─── Proof belongs to the invoice ─────────────────────────────────────────────

// Approving a proof settles the invoice named in the URL. If the two are not
// related, a mistyped id credits money to the wrong participant while both
// records still look plausible.
func TestReviewProof_RefusesProofFromAnotherInvoice(t *testing.T) {
	h := newHarness(t)
	seedPayable(h)
	h.Invoices.Seed(domainInvoice.Invoice{
		ID: "invoice-2", InvoiceNumber: "INV-202608-0002",
		ParticipantID: "participant-1", BatchID: "batch-1",
		Amount: 5000000, Status: "menunggu_bayar", IssuedBy: "user-admin",
		DueDate: time.Now().Add(7 * 24 * time.Hour),
	})
	h.Proofs.Seed(domainInvoice.PaymentProof{
		ID: "proof-untuk-invoice-2", InvoiceID: "invoice-2",
		FilePath: "participant-1/transfer.jpg", AmountClaimed: 5000000, Status: "menunggu",
	})

	h.as("admin").PATCH("/api/v1/admin/invoices/invoice-1/proofs/proof-untuk-invoice-2/review",
		map[string]string{"status": "disetujui", "notes": ""}).
		expectCode(http.StatusUnprocessableEntity)

	inv, _ := h.Invoices.GetByID(t.Context(), "invoice-1")
	if inv.Status == "lunas" {
		t.Error("invoice diselesaikan oleh bukti bayar milik invoice lain")
	}
}
