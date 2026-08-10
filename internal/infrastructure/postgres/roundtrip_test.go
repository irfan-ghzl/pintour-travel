package postgres

// Round-trip tests: write through the repository, read back through the
// repository, and check that what comes out is what went in.
//
// These catch the class of defect that only appears where Go meets SQL — a
// column left out of a SELECT so a field silently reads empty, a scan order that
// no longer matches the projection, a filter that never reaches the WHERE
// clause. None of it is visible from a fake, and none of it needs a scenario to
// find: it needs the real schema.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/airport"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/chatbot"
	domainDocument "github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainNotification "github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

func TestUserRepo_RoundTrip(t *testing.T) {
	db := migratedDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()
	email := fmt.Sprintf("staf%d@uji.local", time.Now().UnixNano())

	u := &domainUser.User{
		Name: "Staf Uji", Email: email, Password: "hash", Role: "konsultan",
		Phone: "628123123123", IsActive: true,
	}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == "" {
		t.Fatal("create did not return an id")
	}

	got, err := repo.GetByEmail(ctx, email)
	if err != nil || got == nil {
		t.Fatalf("get by email: %v", err)
	}
	if got.Name != u.Name || got.Role != u.Role || got.Phone != u.Phone {
		t.Errorf("read back %+v, want name/role/phone to match what was written", got)
	}
	if got.Password != "hash" {
		t.Error("password hash did not survive the round trip — login would always fail")
	}

	got.Name = "Staf Diubah"
	got.Role = "admin"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	after, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if after.Name != "Staf Diubah" || after.Role != "admin" {
		t.Errorf("update did not stick: %+v", after)
	}

	admins, err := repo.ListByRole(ctx, "admin")
	if err != nil {
		t.Fatalf("list by role: %v", err)
	}
	if len(admins) == 0 {
		t.Error("the account just promoted to admin is not in the admin listing")
	}

	if err := repo.Deactivate(ctx, u.ID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	// A deactivated account must drop out of the role listing: the last-super-admin
	// guard counts through it, and an inactive account cannot hold the seat.
	admins, err = repo.ListByRole(ctx, "admin")
	if err != nil {
		t.Fatalf("list by role after deactivate: %v", err)
	}
	for _, a := range admins {
		if a.ID == u.ID {
			t.Error("a deactivated account is still listed as an active admin")
		}
	}

	if missing, err := repo.GetByEmail(ctx, "tidak-ada@uji.local"); err != nil || missing != nil {
		t.Errorf("unknown email = (%v, %v), want (nil, nil) — Login branches on exactly this",
			missing, err)
	}
}

func TestPackageRepo_RoundTripAndFilters(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewPackageRepo(db)
	slug := fmt.Sprintf("paket-baru-%d", time.Now().UnixNano())

	p := &domainPkg.Package{
		Name: "Paket Baru", Slug: slug, Destination: "Jepang", Category: "reguler",
		DurationDays: 7, BasePrice: 18000000, IsActive: true, CreatedBy: f.AdminID,
		Description: "Deskripsi uji",
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("get by slug: %v", err)
	}
	if got.Name != p.Name || got.BasePrice != p.BasePrice || got.DurationDays != p.DurationDays {
		t.Errorf("read back %+v, want the written values", got)
	}

	got.BasePrice = 19500000
	got.IsActive = false
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	after, err := repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if after.BasePrice != 19500000 || after.IsActive {
		t.Errorf("update did not stick: %+v", after)
	}

	// Filters have to reach the WHERE clause, not just the struct.
	active := true
	live, _, err := repo.List(ctx, domainPkg.Filter{IsActive: &active, Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	for _, item := range live {
		if item.ID == p.ID {
			t.Error("a deactivated package is still listed as active")
		}
	}

	dest := "Arab"
	byDest, _, err := repo.List(ctx, domainPkg.Filter{Destination: &dest, Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("list by destination: %v", err)
	}
	if len(byDest) == 0 {
		t.Error("destination filter matched nothing — the seeded package is in Arab Saudi")
	}

	if err := repo.Delete(ctx, p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetBySlug(ctx, slug); err == nil {
		t.Error("a deleted package is still reachable by slug")
	}
}

func TestPackageBatchAndImageRepo_RoundTrip(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	batches := NewPackageBatchRepo(db)
	images := NewPackageImageRepo(db)

	b := &domainPkg.PackageBatch{
		PackageID: f.PackageID, DepartureDate: calendar.Today().AddDays(90),
		ReturnDate: calendar.Today().AddDays(99), Quota: 25,
		PriceSingle: 30000000, PriceDouble: 26000000, PriceTriple: 24000000,
		Status: "tersedia",
	}
	if err := batches.Create(ctx, b); err != nil {
		t.Fatalf("create batch: %v", err)
	}
	got, err := batches.GetByID(ctx, b.ID)
	if err != nil {
		t.Fatalf("get batch: %v", err)
	}
	if got.Quota != 25 || got.PriceDouble != 26000000 {
		t.Errorf("batch read back as %+v", got)
	}

	got.Status = "penuh"
	got.Quota = 30
	if err := batches.Update(ctx, got); err != nil {
		t.Fatalf("update batch: %v", err)
	}
	after, err := batches.GetByID(ctx, b.ID)
	if err != nil {
		t.Fatalf("get batch after update: %v", err)
	}
	if after.Status != "penuh" || after.Quota != 30 {
		t.Errorf("batch update did not stick: %+v", after)
	}

	list, err := batches.ListByPackage(ctx, f.PackageID)
	if err != nil {
		t.Fatalf("list batches: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("package has %d batches, want the seeded one plus the new one", len(list))
	}

	img := &domainPkg.PackageImage{PackageID: f.PackageID, FilePath: "paket/1.jpg", SortOrder: 1}
	if err := images.Add(ctx, img); err != nil {
		t.Fatalf("add image: %v", err)
	}
	imgs, err := images.ListByPackage(ctx, f.PackageID)
	if err != nil || len(imgs) != 1 {
		t.Fatalf("list images = %d, %v; want 1", len(imgs), err)
	}
	if err := images.Delete(ctx, img.ID); err != nil {
		t.Fatalf("delete image: %v", err)
	}
	if imgs, _ = images.ListByPackage(ctx, f.PackageID); len(imgs) != 0 {
		t.Error("a deleted image is still listed")
	}
}

func TestLeadRepo_RoundTripFiltersAndNotes(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewLeadRepo(db)
	notes := NewLeadNoteRepo(db)

	l := &domainLead.Lead{
		Name: "Prospek Baru", Phone: "628999111222", Email: "baru@uji.local",
		PackageID: f.PackageID, BatchID: &f.BatchID, Pax: 3, Message: "tanya harga",
		Source: "meta_ads", Status: "baru", AssignedTo: &f.ConsultantID, ConsentGiven: true,
	}
	if err := repo.Create(ctx, l); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByID(ctx, l.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Pax != 3 || got.Source != "meta_ads" || got.Message != "tanya harga" {
		t.Errorf("read back %+v", got)
	}
	if got.PackageName == "" {
		t.Error("package name was not joined — the CRM listing shows it")
	}
	if got.AssigneeName == nil || *got.AssigneeName == "" {
		t.Error("assignee name was not joined")
	}

	status := "baru"
	byStatus, total, err := repo.List(ctx, domainLead.Filter{Status: &status, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	if total == 0 || len(byStatus) == 0 {
		t.Error("status filter matched nothing")
	}
	for _, item := range byStatus {
		if item.Status != status {
			t.Errorf("status filter returned a %q lead", item.Status)
			break
		}
	}

	search := "Prospek Baru"
	found, _, err := repo.List(ctx, domainLead.Filter{Search: &search, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) == 0 {
		t.Error("search by name matched nothing")
	}

	if err := repo.AssignTo(ctx, l.ID, f.AdminID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if after, _ := repo.GetByID(ctx, l.ID); after.AssignedTo == nil || *after.AssignedTo != f.AdminID {
		t.Error("assignment did not stick")
	}

	if err := repo.MarkConverted(ctx, l.ID, f.AdminID); err == nil {
		t.Error("converting a lead that is not 'deal' was allowed")
	}

	note := &domainLead.Note{LeadID: l.ID, UserID: f.ConsultantID, Note: "sudah ditelepon"}
	if err := notes.Create(ctx, note); err != nil {
		t.Fatalf("create note: %v", err)
	}
	list, err := notes.ListByLead(ctx, l.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list notes = %d, %v; want 1", len(list), err)
	}
	if list[0].Note != "sudah ditelepon" || list[0].UserName == "" {
		t.Errorf("note read back as %+v — the author's name is shown in the activity log", list[0])
	}

	n, err := repo.CountActiveByConsultant(ctx, f.ConsultantID)
	if err != nil {
		t.Fatalf("count by consultant: %v", err)
	}
	if n == 0 {
		t.Error("the consultant holding the seeded leads counts zero — auto-assign balances on this")
	}
}

func TestInvoiceAndProofRepo_RoundTrip(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	invoices := NewInvoiceRepo(db)
	proofs := NewPaymentProofRepo(db)

	inv := &domainInvoice.Invoice{
		InvoiceNumber: fmt.Sprintf("INV-RT-%d", time.Now().UnixNano()%1000000),
		ParticipantID: f.PaxIDs[0], BatchID: f.BatchID, Amount: 12500000,
		DueDate: calendar.Today().AddDays(14), Status: "diterbitkan",
		IssuedBy: f.AdminID, Notes: "catatan uji",
	}
	if err := invoices.Create(ctx, inv); err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	got, err := invoices.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("get invoice: %v", err)
	}
	if got.Amount != 12500000 || got.Notes != "catatan uji" {
		t.Errorf("invoice read back as %+v", got)
	}
	if got.ParticipantName == "" || got.PackageName == "" || got.IssuedByName == "" {
		t.Errorf("joined fields are empty (%q/%q/%q) — the PDF prints all three",
			got.ParticipantName, got.PackageName, got.IssuedByName)
	}

	if err := invoices.SetSnap(ctx, inv.ID, "snap-token", "ORDER-RT-1"); err != nil {
		t.Fatalf("set snap: %v", err)
	}

	pp := &domainInvoice.PaymentProof{
		InvoiceID: inv.ID, FilePath: "bukti/transfer.jpg",
		AmountClaimed: 5000000, Notes: "transfer pertama",
	}
	if err := proofs.Create(ctx, pp); err != nil {
		t.Fatalf("create proof: %v", err)
	}
	if pp.ID == "" {
		t.Fatal("proof create did not return an id — the settle path uses it immediately")
	}
	list, err := proofs.GetByInvoice(ctx, inv.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list proofs = %d, %v; want 1", len(list), err)
	}
	if list[0].Status != "menunggu" {
		t.Errorf("a new proof starts as %q, want menunggu", list[0].Status)
	}

	if err := proofs.Review(ctx, pp.ID, "disetujui", f.AdminID, "cocok"); err != nil {
		t.Fatalf("review proof: %v", err)
	}
	one, err := proofs.GetByID(ctx, pp.ID)
	if err != nil {
		t.Fatalf("get proof: %v", err)
	}
	if one.Status != "disetujui" || one.ReviewNotes != "cocok" || one.ReviewedAt == nil {
		t.Errorf("review did not stick: %+v", one)
	}

	if err := invoices.Confirm(ctx, inv.ID, f.AdminID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	settled, err := invoices.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("get after confirm: %v", err)
	}
	if settled.Status != "lunas" || settled.ConfirmedBy == nil || settled.ConfirmedAt == nil {
		t.Errorf("confirm left the invoice as %+v", settled)
	}

	paid := "lunas"
	byStatus, _, err := invoices.List(ctx, domainInvoice.Filter{Status: &paid, Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	for _, item := range byStatus {
		if item.Status != paid {
			t.Errorf("status filter returned a %q invoice", item.Status)
			break
		}
	}
}

func TestDocumentRepo_RoundTripAndReview(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewDocumentRepo(db)

	d := &domainDocument.Document{
		ParticipantID: f.PaxIDs[0], DocumentType: "ktp",
		FilePath: f.PaxIDs[0] + "/ktp.jpg", FileName: "ktp.jpg", Status: "menunggu",
	}
	if err := repo.Create(ctx, d); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByID(ctx, d.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.FileName != "ktp.jpg" || got.ParticipantID != f.PaxIDs[0] {
		t.Errorf("read back %+v", got)
	}

	if err := repo.Review(ctx, d.ID, "ditolak", f.AdminID, "foto buram"); err != nil {
		t.Fatalf("review: %v", err)
	}
	after, err := repo.GetByID(ctx, d.ID)
	if err != nil {
		t.Fatalf("get after review: %v", err)
	}
	if after.Status != "ditolak" || after.RejectionReason != "foto buram" {
		t.Errorf("review did not stick: %+v", after)
	}

	mine, err := repo.ListByParticipant(ctx, f.PaxIDs[0])
	if err != nil || len(mine) == 0 {
		t.Fatalf("list by participant = %d, %v", len(mine), err)
	}

	status := "ditolak"
	rejected, _, err := repo.List(ctx, domainDocument.Filter{Status: &status, Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	for _, item := range rejected {
		if item.Status != status {
			t.Errorf("status filter returned a %q document", item.Status)
			break
		}
	}

	if err := repo.Delete(ctx, d.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, d.ID); err == nil {
		t.Error("a soft-deleted document is still readable by id")
	}
}

func TestCountryRequirementRepo_RoundTrip(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	repo := NewCountryRequirementRepo(db)

	cr := &domainDocument.CountryRequirement{
		CountryCode: "JP", CountryName: "Jepang", DocumentType: "passport",
		IsRequired: true, Description: "berlaku 6 bulan",
	}
	if err := repo.Create(ctx, cr); err != nil {
		t.Fatalf("create: %v", err)
	}
	byCountry, err := repo.List(ctx, "JP")
	if err != nil || len(byCountry) == 0 {
		t.Fatalf("list JP = %d, %v", len(byCountry), err)
	}

	// An empty country code means "every requirement" — the admin listing.
	all, err := repo.List(ctx, "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) < len(byCountry) {
		t.Error("the global listing returned fewer rows than one country's")
	}

	cr.IsRequired = false
	cr.Description = "opsional"
	if err := repo.Update(ctx, cr); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, _ := repo.List(ctx, "JP")
	for _, item := range updated {
		if item.ID == cr.ID && (item.IsRequired || item.Description != "opsional") {
			t.Errorf("update did not stick: %+v", item)
		}
	}

	if err := repo.Delete(ctx, cr.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestAirportRepo_ChecklistLifecycle(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewAirportRepo(db)

	if err := repo.InitForBatch(ctx, f.BatchID); err != nil {
		t.Fatalf("init: %v", err)
	}
	before, err := repo.ListByBatch(ctx, airport.Filter{BatchID: f.BatchID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(before) == 0 {
		t.Fatal("init produced no checklist rows")
	}

	// Init is idempotent: running it twice must not double the rows.
	if err := repo.InitForBatch(ctx, f.BatchID); err != nil {
		t.Fatalf("init twice: %v", err)
	}
	again, _ := repo.ListByBatch(ctx, airport.Filter{BatchID: f.BatchID})
	if len(again) != len(before) {
		t.Errorf("init doubled the checklist: %d then %d", len(before), len(again))
	}

	pax := before[0].ParticipantID
	if err := repo.UpdateBaggage(ctx, pax, f.BatchID, f.AdminID); err != nil {
		t.Fatalf("baggage: %v", err)
	}
	if err := repo.UpdateTicket(ctx, pax, f.BatchID, f.AdminID); err != nil {
		t.Fatalf("ticket: %v", err)
	}
	if err := repo.UpdatePassport(ctx, pax, f.BatchID, f.AdminID); err != nil {
		t.Fatalf("passport: %v", err)
	}

	done := "done"
	completed, err := repo.ListByBatch(ctx, airport.Filter{BatchID: f.BatchID, Status: &done})
	if err != nil {
		t.Fatalf("list done: %v", err)
	}
	if len(completed) != 1 {
		t.Errorf("%d participants are complete, want exactly the one just processed", len(completed))
	}
	if completed[0].HandledByName == nil || *completed[0].HandledByName == "" {
		t.Error("the handler's name was not joined — FR-AIR-03 records who returned the passport")
	}
	if completed[0].PassportReturnedAt == nil {
		t.Error("passport return has no timestamp — FR-AIR-03 asks for one")
	}

	progress, err := repo.GetBatchProgress(ctx, f.BatchID)
	if err != nil {
		t.Fatalf("progress: %v", err)
	}
	if progress.TotalPax != len(before) || progress.DoneCount != 1 {
		t.Errorf("progress = %+v, want %d total and 1 done", progress, len(before))
	}
	if progress.PendingCount != progress.TotalPax-progress.DoneCount {
		t.Errorf("progress does not add up: %+v", progress)
	}
	if progress.BatchID != f.BatchID {
		t.Errorf("progress reports batch %q, want %q", progress.BatchID, f.BatchID)
	}
}

func TestTourLeaderRepo_RoundTrip(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewTourLeaderRepo(db)

	tl := &domainUser.TourLeader{
		UserID: f.AdminID, Bio: "berpengalaman 10 tahun",
		PhotoPath: "tl/foto.jpg", ExperienceYears: 10, EmergencyPhone: "628700000001",
	}
	if err := repo.Create(ctx, tl); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByUserID(ctx, f.AdminID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Bio != tl.Bio || got.ExperienceYears != 10 {
		t.Errorf("read back %+v", got)
	}

	got.Bio = "berpengalaman 12 tahun"
	got.ExperienceYears = 12
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	after, _ := repo.GetByUserID(ctx, f.AdminID)
	if after.ExperienceYears != 12 {
		t.Errorf("update did not stick: %+v", after)
	}

	list, err := repo.List(ctx)
	if err != nil || len(list) == 0 {
		t.Fatalf("list = %d, %v", len(list), err)
	}
}

func TestPortalUserRepo_RoundTrip(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	repo := NewPortalUserRepo(db)
	phone := fmt.Sprintf("6285%08d", time.Now().UnixNano()%100000000)

	u := &portaluser.PortalUser{
		Phone: phone, PasswordHash: "hash", Name: "Pelanggan", Email: "pelanggan@uji.local",
	}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("create: %v", err)
	}
	byPhone, err := repo.GetByPhone(ctx, phone)
	if err != nil {
		t.Fatalf("get by phone: %v", err)
	}
	if byPhone.PasswordHash != "hash" || byPhone.Name != "Pelanggan" {
		t.Errorf("read back %+v", byPhone)
	}
	byID, err := repo.GetByID(ctx, u.ID)
	if err != nil || byID.Phone != phone {
		t.Fatalf("get by id = %+v, %v", byID, err)
	}
	if _, err := repo.GetByPhone(ctx, "628000000000"); err == nil {
		t.Error("an unknown phone reported success — convert branches on this to decide new vs returning")
	}
}

func TestNotificationRepo_RoundTrip(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	repo := NewNotificationRepo(db)

	refID := f.InvoiceIDs[0]
	refType := "invoice"
	n := &domainNotification.WANotification{
		RecipientPhone: "628111000001", RecipientName: "Peserta",
		MessageType: domainNotification.TypeInvoiceSent, MessageContent: "isi pesan",
		ReferenceID: &refID, ReferenceType: &refType,
	}
	if err := repo.Create(ctx, n); err != nil {
		t.Fatalf("create: %v", err)
	}
	if n.ID == "" {
		t.Fatal("create did not return an id — UpdateStatus needs it")
	}
	if err := repo.UpdateStatus(ctx, n.ID, "sent", map[string]any{"ok": true}); err != nil {
		t.Fatalf("update status: %v", err)
	}
	list, err := repo.ListByReference(ctx, refID, refType)
	if err != nil || len(list) == 0 {
		t.Fatalf("list by reference = %d, %v", len(list), err)
	}
	if list[0].Status != "sent" {
		t.Errorf("status = %q, want sent", list[0].Status)
	}
}

func TestChatbotRepo_RoundTrip(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	repo := NewChatbotRepo(db)
	phone := "628222000111"

	for i, role := range []string{"user", "assistant", "user", "assistant"} {
		if err := repo.Create(ctx, &chatbot.Log{
			Phone: phone, Role: role, Message: fmt.Sprintf("pesan %d", i),
		}); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	recent, err := repo.RecentByPhone(ctx, phone, 3)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent) != 3 {
		t.Errorf("recent returned %d, want the 3 most recent", len(recent))
	}
	// Oldest first: the model is handed a conversation, not a reversed one.
	if len(recent) == 3 && recent[0].Message == "pesan 3" {
		t.Error("history came back newest-first — the model would read the conversation backwards")
	}

	all, err := repo.ListByPhone(ctx, phone)
	if err != nil || len(all) != 4 {
		t.Fatalf("list by phone = %d, %v; want 4", len(all), err)
	}

	convos, total, err := repo.ListConversations(ctx, chatbot.Filter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if total == 0 || len(convos) == 0 {
		t.Fatal("conversation listing is empty")
	}
	if convos[0].MessageCount != 4 {
		t.Errorf("conversation reports %d messages, want 4", convos[0].MessageCount)
	}
}

func TestOCRResultRepo_RoundTrip(t *testing.T) {
	db := migratedDB(t)
	f := seedBulk(t, db)
	ctx := context.Background()
	docs := NewDocumentRepo(db)
	repo := NewOCRResultRepo(db)

	d := &domainDocument.Document{
		ParticipantID: f.PaxIDs[0], DocumentType: "passport",
		FilePath: "p/paspor.jpg", FileName: "paspor.jpg", Status: "menunggu",
	}
	if err := docs.Create(ctx, d); err != nil {
		t.Fatalf("create document: %v", err)
	}

	if err := repo.Create(ctx, &domainDocument.OCRResult{
		DocumentID:    d.ID,
		ExtractedData: []byte(`{"expiry_date":"2030-05-01","document_number":"A1234567"}`),
		Confidence:    0.91, ValidationPassed: true, ValidationNotes: "ok",
	}); err != nil {
		t.Fatalf("create ocr result: %v", err)
	}

	got, err := repo.GetByDocument(ctx, d.ID)
	if err != nil {
		t.Fatalf("get by document: %v", err)
	}
	if got.Confidence != 0.91 || !got.ValidationPassed {
		t.Errorf("read back %+v", got)
	}

	expiry, err := repo.LatestPassportExpiry(ctx, f.PaxIDs[0])
	if err != nil {
		t.Fatalf("latest expiry: %v", err)
	}
	if expiry != "2030-05-01" {
		t.Errorf("expiry = %q, want the date stored in extracted_data — the portal banner reads it", expiry)
	}

	none, err := repo.LatestPassportExpiry(ctx, f.PaxIDs[1])
	if err != nil {
		t.Errorf("a participant with no OCR result returned an error: %v", err)
	}
	if none != "" {
		t.Errorf("expiry = %q for a participant with no scan, want empty", none)
	}
}
