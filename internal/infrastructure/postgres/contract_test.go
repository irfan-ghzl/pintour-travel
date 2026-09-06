package postgres

// The guarantees the schema makes, checked against the schema.
//
// Each test here stands behind a specific claim a ticket made and a fake could
// not settle: the audit row is written by the same statement as the status, the
// listings hide soft-deleted rows, an invoice number is never handed out twice,
// a gateway notification is claimed by exactly one caller.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainDocument "github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	pkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/privacy"
)

// ─── Tiket 06: jejak audit perubahan status ──────────────────────────────────

// FR-CRM-02 asks for the actor, and the actor is the part that used to be
// thrown away: the repository took changedBy and ignored it.
func TestLeadRepo_StatusChangeRecordsWhoMadeIt(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	repo := NewLeadRepo(db)
	ctx := context.Background()

	leadID := f.LeadIDs[0]
	if err := repo.UpdateStatus(ctx, leadID, "dihubungi", f.ConsultantID); err != nil {
		t.Fatalf("first change: %v", err)
	}
	if err := repo.UpdateStatus(ctx, leadID, "konsultasi", f.AdminID); err != nil {
		t.Fatalf("second change: %v", err)
	}

	history, err := repo.ListStatusHistory(ctx, leadID)
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("history has %d rows, want 2", len(history))
	}
	if history[0].ToStatus != "dihubungi" || history[0].ChangedBy != f.ConsultantID {
		t.Errorf("first row = %+v, want dihubungi by the consultant", history[0])
	}
	if history[1].FromStatus != "dihubungi" || history[1].ToStatus != "konsultasi" {
		t.Errorf("second row lost the transition: %+v", history[1])
	}
	if history[1].ChangedBy != f.AdminID {
		t.Errorf("second row actor = %q, want the admin — two different people must not collapse into one",
			history[1].ChangedBy)
	}
}

// The status and its history row are written by one statement, so a status that
// changed always has a row explaining it. A lead that does not exist changes
// nothing and writes nothing.
func TestLeadRepo_UnknownLeadWritesNoHistory(t *testing.T) {
	db := migratedDB(t)
	seedBulk(t, db)
	repo := NewLeadRepo(db)

	before := count(t, db, `SELECT COUNT(*) FROM lead_status_history`)
	err := repo.UpdateStatus(context.Background(),
		"11111111-1111-1111-1111-111111111111", "deal", "")
	if err == nil {
		t.Error("changing the status of a lead that does not exist reported success")
	}
	if after := count(t, db, `SELECT COUNT(*) FROM lead_status_history`); after != before {
		t.Errorf("history grew from %d to %d for a lead that does not exist", before, after)
	}
}

// ─── Tiket 06: hapus lunak ────────────────────────────────────────────────────

// Every listing must agree on which rows exist. The invoice repository was the
// one that did not filter, so a deleted invoice stayed visible in listings, in
// reports, and to the scheduler.
func TestRepos_SoftDeletedRowsStayOutOfListings(t *testing.T) {
	db := migratedDB(t)
	seedBulk(t, db)
	ctx := context.Background()

	t.Run("invoice", func(t *testing.T) {
		live := count(t, db, `SELECT COUNT(*) FROM invoices WHERE deleted_at IS NULL`)
		deleted := count(t, db, `SELECT COUNT(*) FROM invoices WHERE deleted_at IS NOT NULL`)
		if deleted == 0 {
			t.Fatal("fixture has no soft-deleted invoices — the test would pass vacuously")
		}
		_, total, err := NewInvoiceRepo(db).List(ctx, domainInvoice.Filter{Page: 1, PerPage: 10})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != live {
			t.Errorf("total = %d, want %d live invoices (%d are soft-deleted)", total, live, deleted)
		}
	})

	t.Run("lead", func(t *testing.T) {
		live := count(t, db, `SELECT COUNT(*) FROM leads WHERE deleted_at IS NULL`)
		_, total, err := NewLeadRepo(db).List(ctx, domainLead.Filter{Page: 1, PerPage: 10})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != live {
			t.Errorf("total = %d, want %d live leads", total, live)
		}
	})

	t.Run("participant", func(t *testing.T) {
		live := count(t, db, `SELECT COUNT(*) FROM participants WHERE deleted_at IS NULL`)
		_, total, err := NewParticipantRepo(db).List(ctx, domainParticipant.Filter{Page: 1, PerPage: 10})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != live {
			t.Errorf("total = %d, want %d live participants", total, live)
		}
	})

	t.Run("document", func(t *testing.T) {
		live := count(t, db, `SELECT COUNT(*) FROM documents WHERE deleted_at IS NULL`)
		_, total, err := NewDocumentRepo(db).List(ctx, domainDocument.Filter{Page: 1, PerPage: 10})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != live {
			t.Errorf("total = %d, want %d live documents", total, live)
		}
	})
}

// ─── Tiket 10: paginasi ───────────────────────────────────────────────────────

// The document listing had no LIMIT at all: the admin dashboard fetched every
// row in the system to display a count. At fixture scale that difference is
// visible; at eight rows it is not.
func TestDocumentRepo_ListPaginates(t *testing.T) {
	db := migratedDB(t)
	seedBulk(t, db)
	ctx := context.Background()
	repo := NewDocumentRepo(db)

	page1, total, err := repo.List(ctx, domainDocument.Filter{Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1) != 20 {
		t.Errorf("page 1 returned %d rows, want 20 — the query is not bounded", len(page1))
	}
	if total <= 20 {
		t.Fatalf("fixture too small to prove pagination: total = %d", total)
	}

	page2, _, err := repo.List(ctx, domainDocument.Filter{Page: 2, PerPage: 20})
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	seen := map[string]bool{}
	for _, d := range page1 {
		seen[d.ID] = true
	}
	for _, d := range page2 {
		if seen[d.ID] {
			t.Errorf("document %s appears on both pages — the offset is wrong", d.ID)
			break
		}
	}
}

// ─── Tiket 07: penomoran invoice ─────────────────────────────────────────────

// §13.7: a number is never reused, including after a soft delete. The sequence
// is derived from existing numbers, so a deleted invoice must still occupy its
// number — otherwise two invoices carry the same one and the second insert dies
// on the unique index.
func TestInvoiceRepo_NextSequenceNeverReusesANumber(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewInvoiceRepo(db)

	yearMonth := time.Now().Format("200601")
	first, err := repo.NextSequence(ctx, yearMonth)
	if err != nil {
		t.Fatalf("next sequence: %v", err)
	}

	inv := &domainInvoice.Invoice{
		InvoiceNumber: fmt.Sprintf("INV-%s-%04d", yearMonth, first),
		ParticipantID: f.PaxIDs[0], BatchID: f.BatchID, Amount: 1000000,
		DueDate: calendar.Today().AddDays(7), Status: "diterbitkan", IssuedBy: f.AdminID,
	}
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("create: %v", err)
	}
	second, err := repo.NextSequence(ctx, yearMonth)
	if err != nil {
		t.Fatalf("next sequence after create: %v", err)
	}
	if second <= first {
		t.Fatalf("sequence did not advance: %d then %d", first, second)
	}

	// Soft-delete it and ask again: the number must stay taken.
	if _, err := db.ExecContext(ctx,
		`UPDATE invoices SET deleted_at = NOW() WHERE id = $1`, inv.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	third, err := repo.NextSequence(ctx, yearMonth)
	if err != nil {
		t.Fatalf("next sequence after delete: %v", err)
	}
	if third < second {
		t.Errorf("sequence went backwards after a soft delete (%d then %d) — the number would be reused",
			second, third)
	}
}

// ─── Tiket 07: idempotensi notifikasi gateway ────────────────────────────────

// The claim is what stops one payment being counted twice. Under concurrent
// delivery — which is what a gateway retry storm looks like — exactly one caller
// may win.
func TestGatewayRepo_NotificationIsClaimedExactlyOnce(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewGatewayOrderRepo(db)

	orderID := "ORDER-CLAIM-1"
	if err := repo.Create(ctx, &domainInvoice.GatewayOrder{
		InvoiceID: f.InvoiceIDs[0], OrderID: orderID, SnapToken: "tok",
	}); err != nil {
		t.Fatalf("create order: %v", err)
	}

	notification := domainInvoice.GatewayNotification{
		InvoiceID: f.InvoiceIDs[0], OrderID: orderID,
		TransactionID: "trx-1", TransactionStatus: "settlement",
	}

	const callers = 8
	var wg sync.WaitGroup
	claims := make([]bool, callers)
	errs := make([]error, callers)
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func(i int) {
			defer wg.Done()
			claims[i], errs[i] = repo.ClaimNotification(ctx, notification)
		}(i)
	}
	wg.Wait()

	won := 0
	for i := range claims {
		if errs[i] != nil {
			t.Errorf("caller %d failed: %v", i, errs[i])
		}
		if claims[i] {
			won++
		}
	}
	if won != 1 {
		t.Errorf("%d of %d callers claimed the same notification, want exactly 1 — "+
			"each extra claim is one duplicate payment proof", won, callers)
	}
}

// A different notification for the same order is a different payment and must
// still be claimable — the key is the notification, not the order.
func TestGatewayRepo_ASecondPaymentOnTheSameOrderIsStillClaimable(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewGatewayOrderRepo(db)

	orderID := "ORDER-CLAIM-2"
	if err := repo.Create(ctx, &domainInvoice.GatewayOrder{
		InvoiceID: f.InvoiceIDs[1], OrderID: orderID,
	}); err != nil {
		t.Fatalf("create order: %v", err)
	}

	first, err := repo.ClaimNotification(ctx, domainInvoice.GatewayNotification{
		InvoiceID: f.InvoiceIDs[1], OrderID: orderID,
		TransactionID: "trx-dp", TransactionStatus: "settlement"})
	if err != nil || !first {
		t.Fatalf("first claim = %v, %v; want true", first, err)
	}
	second, err := repo.ClaimNotification(ctx, domainInvoice.GatewayNotification{
		InvoiceID: f.InvoiceIDs[1], OrderID: orderID,
		TransactionID: "trx-pelunasan", TransactionStatus: "settlement"})
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if !second {
		t.Error("a settlement of a different amount was treated as a repeat — that payment is lost")
	}
}

// ─── Tiket 07: satu invoice, banyak sesi pembayaran ──────────────────────────

// The order id used to be overwritten on the invoice, so a participant who
// reloaded the payment page and then paid in the first tab sent a notification
// naming an order the system no longer knew.
func TestGatewayRepo_EverySessionStaysResolvable(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewGatewayOrderRepo(db)

	invoiceID := f.InvoiceIDs[2]
	orders := []string{"SESI-1", "SESI-2", "SESI-3"}
	for _, o := range orders {
		if err := repo.Create(ctx, &domainInvoice.GatewayOrder{
			InvoiceID: invoiceID, OrderID: o, SnapToken: "tok-" + o,
		}); err != nil {
			t.Fatalf("create %s: %v", o, err)
		}
	}
	for _, o := range orders {
		got, err := repo.FindInvoiceIDByOrder(ctx, o)
		if err != nil {
			t.Errorf("resolve %s: %v", o, err)
			continue
		}
		if got != invoiceID {
			t.Errorf("order %s resolved to %s, want %s — an earlier session was lost", o, got, invoiceID)
		}
	}
}

// ─── Tiket 04 / v3.0: identitas portal mencakup banyak tur ───────────────────

// A returning customer has one account and several tours. The listing matches on
// portal_user_id or phone, which is what makes an unlinked legacy row still
// theirs — and what makes the ownership check on private files correct.
func TestParticipantRepo_ListByPortalUserSpansEveryTour(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewParticipantRepo(db)

	var puID, phone string
	if err := db.QueryRowContext(ctx, `
		SELECT pu.id, pu.phone FROM portal_users pu
		JOIN participants p ON p.portal_user_id = pu.id
		WHERE p.deleted_at IS NULL LIMIT 1`).Scan(&puID, &phone); err != nil {
		t.Fatalf("pick a portal identity: %v", err)
	}

	// Give the same identity a second tour, the way a second booking would.
	var secondBatch string
	must(t, db.QueryRowContext(ctx, `
		INSERT INTO package_batches (package_id, departure_date, return_date, quota,
			price_single, price_double, price_triple, status)
		VALUES ($1, CURRENT_DATE - 200, CURRENT_DATE - 190, 40, 1, 1, 1, 'ditutup')
		RETURNING id`, f.PackageID).Scan(&secondBatch))
	must(t, exec(db, ctx, `
		INSERT INTO participants (portal_user_id, batch_id, name, phone, email,
			room_type, portal_password, is_active)
		VALUES ($1, $2, 'Tur Lampau', $3, 'lampau@uji.local', 'double', 'x', true)`,
		puID, secondBatch, phone))

	trips, err := repo.ListByPortalUser(ctx, puID, phone)
	if err != nil {
		t.Fatalf("list trips: %v", err)
	}
	if len(trips) < 2 {
		t.Fatalf("identity has %d tours, want at least 2 — history is missing", len(trips))
	}
	// Newest departure first: the portal picks trips[0] as the default context.
	for i := 1; i < len(trips); i++ {
		if trips[i-1].BatchDepartureDate == nil || trips[i].BatchDepartureDate == nil {
			continue
		}
		if trips[i-1].BatchDepartureDate.Before(*trips[i].BatchDepartureDate) {
			t.Errorf("tours are not ordered newest-departure-first at index %d", i)
			break
		}
	}
}

// A consultant sees only participants converted from their own leads (§5.3).
func TestParticipantRepo_ConsultantScopeOnlyReturnsTheirOwn(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()

	var otherConsultant string
	must(t, db.QueryRowContext(ctx, `
		INSERT INTO users (name, email, password, role)
		VALUES ('Konsultan Lain', $1, 'x', 'konsultan') RETURNING id`,
		fmt.Sprintf("lain%d@uji.local", time.Now().UnixNano())).Scan(&otherConsultant))

	_, mine, err := NewParticipantRepo(db).List(ctx, domainParticipant.Filter{
		AssignedTo: &f.ConsultantID, Page: 1, PerPage: 10,
	})
	if err != nil {
		t.Fatalf("list scoped to the seeding consultant: %v", err)
	}
	_, theirs, err := NewParticipantRepo(db).List(ctx, domainParticipant.Filter{
		AssignedTo: &otherConsultant, Page: 1, PerPage: 10,
	})
	if err != nil {
		t.Fatalf("list scoped to a consultant with no leads: %v", err)
	}
	if mine == 0 {
		t.Error("the seeding consultant sees none of their own participants")
	}
	if theirs != 0 {
		t.Errorf("a consultant with no leads sees %d participants", theirs)
	}
}

// ─── §25.5: permintaan penghapusan data ──────────────────────────────────────

// One open request per participant is enforced by a partial unique index, not by
// the handler — so it holds even when two requests race.
func TestDeletionRequestRepo_OnlyOneOpenRequestPerParticipant(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	repo := NewDeletionRequestRepo(db)
	ctx := context.Background()

	first := &privacy.DeletionRequest{ParticipantID: f.PaxIDs[0], Reason: "alasan pertama"}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first request: %v", err)
	}
	second := &privacy.DeletionRequest{ParticipantID: f.PaxIDs[0], Reason: "alasan kedua"}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("second request: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("asking twice produced two requests (%s and %s) — the admin queue would hold the same person twice",
			first.ID, second.ID)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM data_deletion_requests WHERE participant_id=$1`,
		f.PaxIDs[0]); n != 1 {
		t.Errorf("%d rows stored, want 1", n)
	}
}

// Anonymisation has to leave the row and its invoices standing (§25.4 keeps
// participant data for statistics; an invoice is a financial record) while the
// fields that make the row a person are gone.
func TestDeletionRequestRepo_AnonymiseKeepsTheRowAndDropsThePerson(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	repo := NewDeletionRequestRepo(db)
	ctx := context.Background()

	paxID := f.PaxIDs[0]
	var beforeName, beforePhone string
	must(t, db.QueryRowContext(ctx,
		`SELECT name, phone FROM participants WHERE id=$1`, paxID).Scan(&beforeName, &beforePhone))
	invoicesBefore := count(t, db, `SELECT COUNT(*) FROM invoices WHERE participant_id=$1`, paxID)
	docsBefore := count(t, db,
		`SELECT COUNT(*) FROM documents WHERE participant_id=$1 AND deleted_at IS NULL`, paxID)
	if docsBefore == 0 {
		t.Fatal("fixture participant has no documents — the erasure would be untested")
	}

	req := &privacy.DeletionRequest{ParticipantID: paxID}
	must(t, repo.Create(ctx, req))
	if err := repo.Anonymise(ctx, req.ID, f.AdminID, "tidak ada keberangkatan aktif"); err != nil {
		t.Fatalf("anonymise: %v", err)
	}

	var name, phone string
	var email, nik sql.NullString
	var anonymisedAt sql.NullTime
	must(t, db.QueryRowContext(ctx, `
		SELECT name, phone, email, nik, anonymized_at FROM participants WHERE id=$1`, paxID).
		Scan(&name, &phone, &email, &nik, &anonymisedAt))

	if name == beforeName {
		t.Error("the participant's name survived the erasure")
	}
	if phone == beforePhone {
		t.Error("the participant's phone survived — it is the portal login identity")
	}
	if email.Valid && email.String != "" {
		t.Errorf("email survived as %q", email.String)
	}
	if nik.Valid && nik.String != "" {
		t.Errorf("NIK survived as %q", nik.String)
	}
	if !anonymisedAt.Valid {
		t.Error("anonymized_at was not stamped — nothing records that this happened")
	}

	if after := count(t, db, `SELECT COUNT(*) FROM participants WHERE id=$1`, paxID); after != 1 {
		t.Error("the participant row was removed — §25.4 anonymises rather than deletes")
	}
	if after := count(t, db, `SELECT COUNT(*) FROM invoices WHERE participant_id=$1`, paxID); after != invoicesBefore {
		t.Errorf("invoices went from %d to %d — a financial record must survive erasure",
			invoicesBefore, after)
	}
	if live := count(t, db,
		`SELECT COUNT(*) FROM documents WHERE participant_id=$1 AND deleted_at IS NULL`, paxID); live != 0 {
		t.Errorf("%d documents are still live — the scans are the sensitive part", live)
	}

	// With the row soft-deleted, every listing has to stop returning it. This is
	// the filter the participant repository was missing.
	if _, err := NewParticipantRepo(db).GetByID(ctx, paxID); err == nil {
		t.Error("an anonymised participant is still readable by id")
	}
	_, total, err := NewParticipantRepo(db).List(ctx, domainParticipant.Filter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	live := count(t, db, `SELECT COUNT(*) FROM participants WHERE deleted_at IS NULL`)
	if total != live {
		t.Errorf("listing counts %d, want %d live participants", total, live)
	}
}

// A decision is taken once. Two admins working the queue must not both succeed.
func TestDeletionRequestRepo_ASecondDecisionIsRefused(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	repo := NewDeletionRequestRepo(db)
	ctx := context.Background()

	req := &privacy.DeletionRequest{ParticipantID: f.PaxIDs[1]}
	must(t, repo.Create(ctx, req))
	if err := repo.Reject(ctx, req.ID, f.AdminID, "keberangkatan H-7"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if err := repo.Anonymise(ctx, req.ID, f.ConsultantID, "berubah pikiran"); !errors.Is(err, privacy.ErrAlreadyProcessed) {
		t.Errorf("second decision returned %v, want ErrAlreadyProcessed", err)
	}

	var name string
	must(t, db.QueryRowContext(ctx, `SELECT name FROM participants WHERE id=$1`, f.PaxIDs[1]).Scan(&name))
	if name == "Peserta Dihapus" {
		t.Error("the refused request anonymised the participant anyway")
	}

	// A closed request frees the participant to ask again later.
	again := &privacy.DeletionRequest{ParticipantID: f.PaxIDs[1], Reason: "keberangkatan sudah lewat"}
	if err := repo.Create(ctx, again); err != nil {
		t.Fatalf("request after a rejection: %v", err)
	}
	if again.ID == req.ID {
		t.Error("a new request reused the closed one — the rejection would look reopened")
	}
}

// The queue is what the 14-day commitment is worked from, so it has to list
// pending requests with who is asking.
func TestDeletionRequestRepo_QueueCarriesTheRequester(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	repo := NewDeletionRequestRepo(db)
	ctx := context.Background()

	req := &privacy.DeletionRequest{ParticipantID: f.PaxIDs[2], Reason: "pindah agen"}
	must(t, repo.Create(ctx, req))

	pending, err := repo.List(ctx, privacy.StatusPending)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("%d pending requests, want 1", len(pending))
	}
	if pending[0].ParticipantName == "" || pending[0].ParticipantPhone == "" {
		t.Errorf("queue entry does not say who is asking: %+v", pending[0])
	}
	if !pending[0].IsOpen() {
		t.Error("a pending request does not report itself as open")
	}

	done, err := repo.List(ctx, privacy.StatusDone)
	if err != nil {
		t.Fatalf("list done: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("%d requests reported as done, want none", len(done))
	}
}

// ─── Tiket 01: daftar keberangkatan lintas paket ─────────────────────────────

// The cross-package batch listing makes three claims a fake cannot settle: the
// head count comes from a subquery over participants, the ordering is expressed
// in SQL against CURRENT_DATE, and both must survive the soft-delete rule the
// rest of the listings follow.
func TestPackageBatchRepo_ListAllOrdersByNearestAndCountsParticipants(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	repo := NewPackageBatchRepo(db)
	ctx := context.Background()

	// A second package, so "lintas paket" is more than a claim about one row.
	var otherPackageID string
	must(t, db.QueryRowContext(ctx, `
		INSERT INTO packages (name, slug, destination, duration_days, base_price, created_by)
		VALUES ('Halal Tour Jepang', $1, 'Jepang', 8, 32000000, $2) RETURNING id`,
		fmt.Sprintf("halal-jepang-%d", time.Now().UnixNano()), f.AdminID).Scan(&otherPackageID))

	// One departure closer than the fixture's (which leaves in 45 days), and one
	// already gone.
	insertBatch := func(packageID string, offsetDays int, status string) string {
		var id string
		must(t, db.QueryRowContext(ctx, `
			INSERT INTO package_batches (package_id, departure_date, return_date, quota,
				price_single, price_double, price_triple, status)
			VALUES ($1, CURRENT_DATE + $2::int, CURRENT_DATE + $2::int + 8, 25, 3, 2, 1, $3)
			RETURNING id`, packageID, offsetDays, status).Scan(&id))
		return id
	}
	soonID := insertBatch(otherPackageID, 5, "penuh")
	pastID := insertBatch(otherPackageID, -30, "ditutup")

	all, total, err := repo.ListAll(ctx, pkg.BatchFilter{Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Fatalf("listing returned %d rows (total %d), want 3", len(all), total)
	}
	if all[0].ID != soonID {
		t.Errorf("first row = %s, want the departure five days out (%s)", all[0].ID, soonID)
	}
	if all[0].PackageName != "Halal Tour Jepang" {
		t.Errorf("package_name = %q, want the joined package name", all[0].PackageName)
	}
	if last := all[len(all)-1].ID; last != pastID {
		t.Errorf("last row = %s, want the departure already gone (%s)", last, pastID)
	}

	// The fixture put every participant on one batch; the two new ones are empty.
	seeded := count(t, db, `SELECT COUNT(*) FROM participants WHERE batch_id=$1 AND deleted_at IS NULL`, f.BatchID)
	for _, b := range all {
		want := 0
		if b.ID == f.BatchID {
			want = seeded
		}
		if b.ParticipantCount == nil {
			t.Errorf("participant_count for %s is absent; the cross-package listing counts", b.ID)
			continue
		}
		if *b.ParticipantCount != want {
			t.Errorf("participant_count for %s = %d, want %d", b.ID, *b.ParticipantCount, want)
		}
	}

	// The per-package listing feeds the public catalogue, so it must not disclose
	// how many seats a departure has sold.
	perPackage, err := repo.ListByPackage(ctx, otherPackageID)
	if err != nil {
		t.Fatalf("list by package: %v", err)
	}
	if len(perPackage) != 2 {
		t.Fatalf("per-package listing returned %d rows, want 2", len(perPackage))
	}
	for _, b := range perPackage {
		if b.ParticipantCount != nil {
			t.Errorf("per-package listing disclosed a head count for %s (%d) — it also serves the public catalogue",
				b.ID, *b.ParticipantCount)
		}
	}

	upcoming, upcomingTotal, err := repo.ListAll(ctx, pkg.BatchFilter{Upcoming: true, Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("list upcoming: %v", err)
	}
	if upcomingTotal != 2 {
		t.Errorf("upcoming total = %d, want 2", upcomingTotal)
	}
	for _, b := range upcoming {
		if b.ID == pastID {
			t.Error("upcoming=true still returned a departure that has left")
		}
	}

	full := "penuh"
	byStatus, _, err := repo.ListAll(ctx, pkg.BatchFilter{Status: &full, Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	if len(byStatus) != 1 || byStatus[0].ID != soonID {
		t.Errorf("status=penuh returned %d rows, want only %s", len(byStatus), soonID)
	}

	search := "jepang"
	byName, _, err := repo.ListAll(ctx, pkg.BatchFilter{Search: &search, Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("search by package name: %v", err)
	}
	if len(byName) != 2 {
		t.Errorf("search=%q returned %d rows, want the two Halal Tour Jepang departures", search, len(byName))
	}
}
