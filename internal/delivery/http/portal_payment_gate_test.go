package httpdelivery

// Portal sebelum lunas (tiket 01).
//
// The chain used to close on itself: create-payment sat behind a portal token,
// the token came only from login, and login refused anyone whose payment was not
// yet confirmed. To pay, a participant had to have paid.
//
// These tests are written from where the participant stands — the status and the
// body of every portal endpoint, on both sides of the confirmation — because
// that is the whole of what changed. The gate did not disappear; it moved from
// the door to the contents.

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainDocument "github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

// The password the WhatsApp credential message would have carried.
const portalPassword = "Xk7mQp2r"

// gateCustomers is the pair the spec asks be treated identically: someone on
// their first tour, and someone whose account already carries a settled one.
// Every case below runs over both, because the difference between them is
// exactly what the old login rule turned into a difference in what they could
// do (story 15).
var gateCustomers = []struct{ name, participantID, phone string }{
	{"peserta pertama kali", "participant-baru", "628700000001"},
	{"pelanggan lama", "participant-lama-2026", "628700000002"},
}

// ─── World ────────────────────────────────────────────────────────────────────

// seedPortalGateWorld puts two customers on the same upcoming departure, both
// unpaid on the tour they log in as:
//
//   - "participant-baru" is a first-timer: one tour, nothing ever paid.
//   - "participant-lama-2026" belongs to a returning customer whose earlier tour
//     ("participant-lama-2019") is settled and long past.
//
// The pair is the point: story 15 asks that the two be treated identically, and
// the old login accepted the returning customer — whose earlier tour left
// is_active set — while refusing the first-timer.
func seedPortalGateWorld(h *harness) {
	h.Packages.Seed(domainPkg.Package{
		ID: "package-1", Name: "Umroh Reguler 9 Hari", Slug: "umroh-reguler-9-hari",
		Destination: "Arab Saudi", Category: "umroh", DurationDays: 9,
		BasePrice: 25000000, IsActive: true,
		Itinerary: json.RawMessage(`[{"day":1,"title":"Tiba di Jeddah"}]`),
	})
	tourLeaderID := "user-tour_leader"
	h.Batches.Seed(domainPkg.PackageBatch{
		ID: "batch-1", PackageID: "package-1", Quota: 40, Status: "tersedia",
		DepartureDate: calendar.Today().AddDays(30), ReturnDate: calendar.Today().AddDays(39),
		PriceDouble: 25000000, TourLeaderID: &tourLeaderID,
		WaGroupLink: "https://chat.whatsapp.com/batch-1",
	})
	h.Batches.Seed(domainPkg.PackageBatch{
		ID: "batch-2019", PackageID: "package-1", Quota: 40, Status: "tersedia",
		DepartureDate: calendar.Today().AddDays(-365), ReturnDate: calendar.Today().AddDays(-356),
		PriceDouble: 25000000,
	})
	h.TourLeaders.Seed(domainUser.TourLeader{
		ID: "tl-1", UserID: tourLeaderID, Name: "Ahmad Rizki",
		Specialization: "Umroh", EmergencyPhone: "628555000001",
	})

	upcoming := calendar.Today().AddDays(30).Time()
	past := calendar.Today().AddDays(-365).Time()

	// The first-timer.
	h.PortalUsers.Seed(portaluser.PortalUser{
		ID: "portal-user-baru", Phone: "628700000001", Name: "Peserta Baru",
		PasswordHash: hashPortalPassword(h, portalPassword),
	})
	seedGateParticipant(h, gateParticipant{
		id: "participant-baru", portalUser: "portal-user-baru", batch: "batch-1",
		name: "Peserta Baru", phone: "628700000001", departure: &upcoming, paid: false,
	})

	// The returning customer: one settled tour behind them, one unpaid ahead.
	h.PortalUsers.Seed(portaluser.PortalUser{
		ID: "portal-user-lama", Phone: "628700000002", Name: "Peserta Lama",
		PasswordHash: hashPortalPassword(h, portalPassword),
	})
	seedGateParticipant(h, gateParticipant{
		id: "participant-lama-2019", portalUser: "portal-user-lama", batch: "batch-2019",
		name: "Peserta Lama", phone: "628700000002", departure: &past, paid: true,
	})
	seedGateParticipant(h, gateParticipant{
		id: "participant-lama-2026", portalUser: "portal-user-lama", batch: "batch-1",
		name: "Peserta Lama", phone: "628700000002", departure: &upcoming, paid: false,
	})
	h.Invoices.Seed(domainInvoice.Invoice{
		ID: "invoice-lama-2019", InvoiceNumber: "INV-202508-0001",
		ParticipantID: "participant-lama-2019", BatchID: "batch-2019",
		Amount: 25000000, Status: "lunas", DueDate: calendar.Today().AddDays(-380),
	})
}

type gateParticipant struct {
	id, portalUser, batch, name, phone string
	departure                          *time.Time
	paid                               bool
}

// seedGateParticipant seeds one tour plus the rows the portal reads for it: an
// unpaid invoice and a document of the participant's own, so the file-signing
// and invoice endpoints have something real to answer with.
func seedGateParticipant(h *harness, g gateParticipant) {
	portalUser := g.portalUser
	h.Participants.Seed(domainParticipant.Participant{
		ID: g.id, PortalUserID: &portalUser, BatchID: g.batch,
		Name: g.name, Phone: g.phone, Email: g.id + "@pintour.test",
		PackageName: "Umroh Reguler 9 Hari", RoomType: "double",
		IsActive: g.paid, BatchDepartureDate: g.departure,
	})
	h.Documents.Seed(domainDocument.Document{
		ID: "document-" + g.id, ParticipantID: g.id, DocumentType: "passport",
		FilePath: g.id + "/passport.jpg", FileName: "passport.jpg", Status: "menunggu",
	})
	if !g.paid {
		h.Invoices.Seed(domainInvoice.Invoice{
			ID: "invoice-" + g.id, InvoiceNumber: "INV-202608-" + g.id,
			ParticipantID: g.id, BatchID: g.batch, Amount: 25000000,
			Status: "diterbitkan", DueDate: calendar.Today().AddDays(7),
		})
	}
}

func hashPortalPassword(h *harness, password string) string {
	h.t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		h.t.Fatalf("hash password portal: %v", err)
	}
	return string(hashed)
}

// portalLogin signs in through the real endpoint.
func portalLogin(h *harness, phone, password string) *response {
	h.t.Helper()
	return h.anonymous().POST("/api/v1/portal/login", map[string]any{
		"phone": phone, "password": password,
	})
}

// asLoggedIn returns a client carrying the token login issued — not one minted
// by the harness. Every content test goes through it, because "the token an
// unpaid participant can actually obtain" is half of what is being asserted.
func asLoggedIn(h *harness, phone string) *client {
	h.t.Helper()
	res := portalLogin(h, phone, portalPassword)
	if res.Code != http.StatusOK {
		h.t.Fatalf("login portal %s gagal: %d %s", phone, res.Code, res.Body)
	}
	token, _ := res.data()["token"].(string)
	return h.anonymous().withHeader("X-Portal-Token", token)
}

// ─── The door ─────────────────────────────────────────────────────────────────

// The defect this ticket exists for: a participant created an hour ago, holding
// the password their WhatsApp message carried, is let in.
func TestPortalLogin_UnpaidParticipantIsLetIn(t *testing.T) {
	h := newHarness(t)
	seedPortalGateWorld(h)

	res := portalLogin(h, "628700000001", portalPassword)
	res.expectCode(http.StatusOK)

	data := res.data()
	if token, _ := data["token"].(string); token == "" {
		t.Errorf("login peserta belum lunas tidak menerbitkan token: %s", res.Body)
	}
	if got := data["participant_id"]; got != "participant-baru" {
		t.Errorf("participant_id = %v, ingin participant-baru", got)
	}
}

// A returning customer lands on the tour they are about to take, not on the one
// they finished last year — otherwise the invoice they are being chased for is
// not the one the portal shows them.
func TestPortalLogin_ReturningCustomerLandsOnTheCurrentTour(t *testing.T) {
	h := newHarness(t)
	seedPortalGateWorld(h)

	res := portalLogin(h, "628700000002", portalPassword)
	res.expectCode(http.StatusOK)

	if got := res.data()["participant_id"]; got != "participant-lama-2026" {
		t.Errorf("participant_id = %v, ingin participant-lama-2026 (perjalanan berjalan)", got)
	}
}

// Authentication failure must not be told apart from authorisation failure, and
// the way to guarantee that is to give both the same answer as an unknown
// number. Otherwise the refusal itself says who has an account.
func TestPortalLogin_RefusalsAreIndistinguishable(t *testing.T) {
	h := newHarness(t)
	seedPortalGateWorld(h)

	answers := map[string]string{}
	record := func(name string, body map[string]any) {
		res := h.anonymous().POST("/api/v1/portal/login", body)
		res.expectCode(http.StatusUnauthorized)
		answers[name] = res.message()
	}
	record("nomor tak ada", map[string]any{"phone": "628999999999", "password": portalPassword})
	// Both kinds of customer, because both kinds of account are refused through
	// their own branch of the login — the central portal identity and the legacy
	// per-participant fallback must not answer differently either.
	for _, customer := range gateCustomers {
		record("password salah/"+customer.name,
			map[string]any{"phone": customer.phone, "password": "salah-sekali"})
	}

	unknown := answers["nomor tak ada"]
	for name, answer := range answers {
		if answer != unknown {
			t.Errorf("penolakan %q berbunyi %q sementara nomor tak dikenal berbunyi %q — "+
				"bedanya memberi tahu siapa yang punya akun", name, answer, unknown)
		}
	}
}

// ─── The contents ─────────────────────────────────────────────────────────────

// portalRequest is one call a participant makes, and the answer they should get.
type portalRequest struct {
	name   string
	method string
	path   string
	body   any
	want   int
}

// openBeforeSettlement is every portal endpoint a participant needs in order to
// pay and to prepare — the spec's "terbuka sebelum lunas" list, one request each.
func openBeforeSettlement(participantID string) []portalRequest {
	return []portalRequest{
		{"profil", http.MethodGet, "/api/v1/portal/me", nil, http.StatusOK},
		{"pembaruan profil", http.MethodPut, "/api/v1/portal/profile",
			map[string]any{"name": "Nama Terkoreksi"}, http.StatusOK},
		{"daftar invoice", http.MethodGet, "/api/v1/portal/invoices", nil, http.StatusOK},
		{"pdf invoice", http.MethodGet, "/api/v1/portal/invoices/invoice-" + participantID + "/pdf",
			nil, http.StatusOK},
		{"unggah bukti transfer", http.MethodPost, "/api/v1/portal/invoices/invoice-" + participantID + "/proofs",
			map[string]any{"file_path": participantID + "/transfer.jpg", "amount_claimed": 5000000},
			http.StatusCreated},
		{"daftar dokumen", http.MethodGet, "/api/v1/portal/documents", nil, http.StatusOK},
		{"unggah dokumen", http.MethodPost, "/api/v1/portal/documents",
			map[string]any{"document_type": "ktp", "file_path": participantID + "/ktp.jpg", "file_name": "ktp.jpg"},
			http.StatusCreated},
		{"syarat dokumen negara", http.MethodGet, "/api/v1/portal/documents/requirements", nil, http.StatusOK},
		{"url bertanda tangan", http.MethodGet,
			"/api/v1/portal/signed-url?type=document&id=document-" + participantID, nil, http.StatusOK},
		{"salinan data §25.5", http.MethodGet, "/api/v1/portal/my-data", nil, http.StatusOK},
		{"pengajuan penghapusan §25.5", http.MethodPost, "/api/v1/portal/account-deletion-request",
			map[string]any{"reason": "batal berangkat"}, http.StatusOK},
		{"riwayat perjalanan", http.MethodGet, "/api/v1/portal/my-trips", nil, http.StatusOK},
		{"prefill konsultasi", http.MethodGet, "/api/v1/portal/consultation-prefill", nil, http.StatusOK},
	}
}

// lockedUntilSettlement is the travel content itself: what a participant who has
// not paid has no claim on yet.
var lockedUntilSettlement = []struct {
	name string
	path string
}{
	{"itinerary", "/api/v1/portal/itinerary"},
	{"briefing", "/api/v1/portal/briefing/pdf"},
	{"kontak tour leader", "/api/v1/portal/batch-leader"},
}

// Everything needed to finish paying and to get the paperwork moving answers
// normally, for a participant who has not paid a rupiah.
func TestPortalUnpaid_EverythingNeededToPayIsOpen(t *testing.T) {
	for _, customer := range gateCustomers {
		t.Run(customer.name, func(t *testing.T) {
			for _, tc := range openBeforeSettlement(customer.participantID) {
				t.Run(tc.name, func(t *testing.T) {
					storage := newSigningStorage(t)
					h := newHarness(t, withStorageServer(storage.URL))
					seedPortalGateWorld(h)

					asLoggedIn(h, customer.phone).do(tc.method, tc.path, tc.body).expectCode(tc.want)
				})
			}
		})
	}
}

// The travel content stays shut, and says why: a participant must be able to
// tell "not yet" from "broken" without asking anyone.
func TestPortalUnpaid_TravelContentIsLocked(t *testing.T) {
	for _, customer := range gateCustomers {
		t.Run(customer.name, func(t *testing.T) {
			h := newHarness(t)
			seedPortalGateWorld(h)

			portal := asLoggedIn(h, customer.phone)

			for _, tc := range lockedUntilSettlement {
				t.Run(tc.name, func(t *testing.T) {
					res := portal.GET(tc.path)
					res.expectCode(http.StatusForbidden)

					var envelope struct {
						Error   string `json:"error"`
						Message string `json:"message"`
					}
					res.decode(&envelope)
					if envelope.Error != "PAYMENT_REQUIRED" {
						t.Errorf("kode = %q, ingin PAYMENT_REQUIRED", envelope.Error)
					}
					if envelope.Message == "" {
						t.Error("penolakan tidak menjelaskan syaratnya")
					}
				})
			}
		})
	}
}

// The itinerary of the tour being paid for is not reachable through the archive
// either — that route serves past trips, and this one has not left yet.
func TestPortalUnpaid_ArchiveDoesNotHandOutTheCurrentItinerary(t *testing.T) {
	h := newHarness(t)
	seedPortalGateWorld(h)

	asLoggedIn(h, "628700000001").
		GET("/api/v1/portal/my-trips/participant-baru/itinerary").
		expectCode(http.StatusForbidden)
}

// Story 16: a tour already paid for stays readable, whatever is outstanding on
// the new one.
func TestPortalUnpaid_EarlierSettledTripStaysReadable(t *testing.T) {
	h := newHarness(t)
	seedPortalGateWorld(h)

	portal := asLoggedIn(h, "628700000002")
	portal.GET("/api/v1/portal/my-trips/participant-lama-2019/itinerary").
		expectCode(http.StatusOK)
	portal.GET("/api/v1/portal/my-trips/participant-lama-2019/invoice-pdf").
		expectCode(http.StatusOK).expectPDF()
}

// A departure that has come and gone does not open anything by itself. The
// spec's archive is "riwayat perjalanan lampau, yang memang sudah dibayar" — a
// trip nobody ever paid for is the one the portal labels "dibatalkan", and it
// carries no more claim on the itinerary than it did the week before.
func TestPortalUnpaid_PastDepartureDoesNotOpenTheContent(t *testing.T) {
	h := newHarness(t)
	seedPortalGateWorld(h)
	departed := *h.Participants.participants["participant-baru"]
	lastYear := calendar.Today().AddDays(-365).Time()
	departed.BatchDepartureDate = &lastYear
	h.Participants.Seed(departed)

	portal := asLoggedIn(h, "628700000001")
	portal.GET("/api/v1/portal/itinerary").expectCode(http.StatusForbidden)
	portal.GET("/api/v1/portal/my-trips/participant-baru/itinerary").
		expectCode(http.StatusForbidden)
}

// Story 14: the gate reads the payment, not the token. Confirming the invoice
// opens the content under the token already in the participant's browser.
func TestPortalSettled_ContentOpensWithoutLoggingInAgain(t *testing.T) {
	for _, customer := range gateCustomers {
		t.Run(customer.name, func(t *testing.T) {
			h := newHarness(t)
			seedPortalGateWorld(h)

			portal := asLoggedIn(h, customer.phone)
			for _, tc := range lockedUntilSettlement {
				portal.GET(tc.path).expectCode(http.StatusForbidden)
			}

			h.as("admin").POST("/api/v1/admin/invoices/invoice-"+customer.participantID+"/confirm", nil).
				expectCode(http.StatusOK)

			for _, tc := range lockedUntilSettlement {
				t.Run(tc.name, func(t *testing.T) {
					// The briefing has its own H-14 window and the departure is thirty
					// days out, so it stays shut — for the other reason, with the other
					// code.
					if tc.name == "briefing" {
						res := portal.GET(tc.path)
						res.expectCode(http.StatusForbidden)
						var envelope struct {
							Error string `json:"error"`
						}
						res.decode(&envelope)
						if envelope.Error != "NOT_YET" {
							t.Errorf("kode = %q, ingin NOT_YET — gerbang H-14 tetap berlaku", envelope.Error)
						}
						return
					}
					portal.GET(tc.path).expectCode(http.StatusOK)
				})
			}
		})
	}
}

// The briefing's two gates stack rather than replace one another: paid but too
// early is NOT_YET, paid and inside the window is the PDF.
func TestPortalSettled_BriefingOpensInsideItsOwnWindow(t *testing.T) {
	h := newHarness(t)
	seedPortalGateWorld(h)
	// Reseeded rather than reached into: the fake owns its rows, and a test that
	// mutates them behind its back is one refactor away from silently doing
	// nothing.
	inWindow := *h.Participants.participants["participant-baru"]
	soon := time.Now().Add(7 * 24 * time.Hour)
	inWindow.BatchDepartureDate = &soon
	inWindow.IsActive = true
	h.Participants.Seed(inWindow)

	asLoggedIn(h, "628700000001").
		GET("/api/v1/portal/briefing/pdf").expectCode(http.StatusOK).expectPDF()
}

// The proof the whole ticket turns on: a participant who has never paid can
// open a payment session. Until now the only participants who could were the
// ones who no longer needed to.
func TestPortalUnpaid_CanStartAnOnlinePayment(t *testing.T) {
	for _, customer := range gateCustomers {
		t.Run(customer.name, func(t *testing.T) {
			stub := newSnapStub(t, "snap-token-pra-lunas")
			h := newHarness(t, withMidtransServer(stub.URL))
			seedPortalGateWorld(h)

			res := asLoggedIn(h, customer.phone).
				POST("/api/v1/portal/invoices/invoice-"+customer.participantID+"/create-payment", nil)
			res.expectCode(http.StatusOK)
			if got := res.data()["snap_token"]; got != "snap-token-pra-lunas" {
				t.Errorf("snap_token = %v, ingin snap-token-pra-lunas", got)
			}
			if len(stub.requests) != 1 {
				t.Fatalf("gateway menerima %d transaksi, ingin 1", len(stub.requests))
			}
		})
	}
}

// ─── What the invoice page reads (tiket 02) ───────────────────────────────────

// A participant who has transferred part of the amount needs two things the
// invoice row alone cannot tell them: what became of the receipt they sent, and
// how much is still owed. Both used to exist only inside the settle logic.
func TestPortalInvoices_CarryProofStatusAndRemainingBalance(t *testing.T) {
	h := newHarness(t)
	seedPortalGateWorld(h)
	h.Proofs.Seed(domainInvoice.PaymentProof{
		ID: "proof-disetujui", InvoiceID: "invoice-participant-baru",
		FilePath: "participant-baru/transfer-1.jpg", AmountClaimed: 10000000,
		Status: "disetujui", UploadedAt: time.Now(),
	})
	h.Proofs.Seed(domainInvoice.PaymentProof{
		ID: "proof-ditolak", InvoiceID: "invoice-participant-baru",
		FilePath: "participant-baru/transfer-2.jpg", AmountClaimed: 5000000,
		Status: "ditolak", ReviewNotes: "Nominal tidak sesuai", UploadedAt: time.Now(),
	})

	res := asLoggedIn(h, "628700000001").GET("/api/v1/portal/invoices")
	res.expectCode(http.StatusOK)

	var envelope struct {
		Data []struct {
			ID        string  `json:"id"`
			Amount    float64 `json:"amount"`
			Paid      float64 `json:"paid_amount"`
			Remaining float64 `json:"remaining_balance"`
			Proofs    []struct {
				ID          string  `json:"id"`
				Status      string  `json:"status"`
				Amount      float64 `json:"amount_claimed"`
				ReviewNotes string  `json:"review_notes"`
			} `json:"proofs"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if len(envelope.Data) != 1 {
		t.Fatalf("invoice terbaca = %d, ingin 1: %s", len(envelope.Data), res.Body)
	}

	inv := envelope.Data[0]
	// Only the approved receipt counts towards the balance — a rejected one is
	// money that never arrived.
	if inv.Paid != 10000000 {
		t.Errorf("paid_amount = %v, ingin 10000000 (hanya bukti disetujui)", inv.Paid)
	}
	if inv.Remaining != 15000000 {
		t.Errorf("remaining_balance = %v, ingin 15000000", inv.Remaining)
	}
	if len(inv.Proofs) != 2 {
		t.Fatalf("bukti terbaca = %d, ingin 2", len(inv.Proofs))
	}
	byID := map[string]string{}
	for _, p := range inv.Proofs {
		byID[p.ID] = p.Status
	}
	if byID["proof-disetujui"] != "disetujui" || byID["proof-ditolak"] != "ditolak" {
		t.Errorf("status bukti = %v, ingin disetujui dan ditolak terbaca apa adanya", byID)
	}
	for _, p := range inv.Proofs {
		if p.ID == "proof-ditolak" && p.ReviewNotes != "Nominal tidak sesuai" {
			t.Errorf("alasan penolakan = %q, ingin terbaca peserta", p.ReviewNotes)
		}
	}
}

// The proofs listed with an invoice are that invoice's own — the endpoint is
// scoped by the token, and one participant's receipts must not appear under
// another's bill.
func TestPortalInvoices_ProofsStayWithTheirOwnInvoice(t *testing.T) {
	h := newHarness(t)
	seedPortalGateWorld(h)
	h.Proofs.Seed(domainInvoice.PaymentProof{
		ID: "proof-orang-lain", InvoiceID: "invoice-participant-lama-2026",
		FilePath: "participant-lama-2026/transfer.jpg", AmountClaimed: 25000000,
		Status: "disetujui", UploadedAt: time.Now(),
	})

	res := asLoggedIn(h, "628700000001").GET("/api/v1/portal/invoices")
	res.expectCode(http.StatusOK)

	var envelope struct {
		Data []struct {
			Paid   float64 `json:"paid_amount"`
			Proofs []struct {
				ID string `json:"id"`
			} `json:"proofs"`
		} `json:"data"`
	}
	res.decode(&envelope)
	for _, inv := range envelope.Data {
		if len(inv.Proofs) != 0 || inv.Paid != 0 {
			t.Errorf("bukti peserta lain ikut terbaca: %+v", inv)
		}
	}
}
