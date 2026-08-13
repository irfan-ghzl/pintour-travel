package httpdelivery

// The two reads behind every batch and participant picker in the admin UI.
//
// Four admin pages used to ask an operator to type a UUID the application never
// prints anywhere. Replacing those boxes with pickers needs one new read —
// batches across every package — and leans on a participant search that already
// existed. What is asserted here is the observable behaviour of both: what comes
// back, in what order, and who is allowed to ask.

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
)

// batchRow is the shape a picker needs from GET /admin/batches: enough to name a
// departure without knowing its id.
type batchRow struct {
	ID            string `json:"id"`
	PackageID     string `json:"package_id"`
	PackageName   string `json:"package_name"`
	DepartureDate string `json:"departure_date"`
	Status        string `json:"status"`
	Quota         int    `json:"quota"`
	// A pointer, mirroring the domain: absent means this read did not count,
	// which is not the same answer as nobody having booked.
	ParticipantCount *int `json:"participant_count"`
}

// batchesOf decodes the paginated envelope of GET /admin/batches.
func (r *response) batchesOf() []batchRow {
	r.t.Helper()
	var envelope struct {
		Data []batchRow `json:"data"`
	}
	r.decode(&envelope)
	return envelope.Data
}

// batchIDs is the order the server answered in, which is what the ordering
// assertions are about.
func batchIDs(rows []batchRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// seedTwoPackagesOfBatches lays out departures over two packages and around
// today: two upcoming for one package, one upcoming and one already gone for the
// other. Every ordering, filtering and aggregation claim below reads from this
// one arrangement.
func seedTwoPackagesOfBatches(h *harness) {
	h.Packages.Seed(domainPkg.Package{
		ID: "package-umroh", Name: "Umroh Plus Istanbul", Slug: "umroh-plus-istanbul",
		Destination: "Arab Saudi", Category: "umroh", DurationDays: 12,
		BasePrice: 29500000, IsActive: true,
	})
	h.Packages.Seed(domainPkg.Package{
		ID: "package-halal", Name: "Halal Tour Jepang", Slug: "halal-tour-jepang",
		Destination: "Jepang", Category: "halal", DurationDays: 8,
		BasePrice: 32000000, IsActive: true,
	})

	today := calendar.Today()
	for _, b := range []domainPkg.PackageBatch{
		{ID: "batch-umroh-jauh", PackageID: "package-umroh", DepartureDate: today.AddDays(60),
			ReturnDate: today.AddDays(72), Quota: 40, Status: "tersedia"},
		{ID: "batch-umroh-dekat", PackageID: "package-umroh", DepartureDate: today.AddDays(7),
			ReturnDate: today.AddDays(19), Quota: 40, Status: "tersedia"},
		{ID: "batch-halal-dekat", PackageID: "package-halal", DepartureDate: today.AddDays(21),
			ReturnDate: today.AddDays(29), Quota: 25, Status: "penuh"},
		{ID: "batch-halal-lampau", PackageID: "package-halal", DepartureDate: today.AddDays(-30),
			ReturnDate: today.AddDays(-22), Quota: 25, Status: "ditutup"},
	} {
		h.Batches.Seed(b)
	}
}

// ─── Tiket 01: GET /admin/batches ─────────────────────────────────────────────

// Listing batches per package is what already existed, and it is exactly what
// the pickers cannot use: three of the four pages have no package in hand.
func TestAdminBatches_ListsDeparturesFromEveryPackage(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)

	rows := h.as("admin").GET("/api/v1/admin/batches").expectCode(http.StatusOK).batchesOf()

	packages := map[string]bool{}
	for _, row := range rows {
		packages[row.PackageID] = true
	}
	if len(packages) != 2 {
		t.Errorf("satu panggilan mengembalikan batch dari %d paket, want 2 (%v)", len(packages), batchIDs(rows))
	}
	if len(rows) != 4 {
		t.Errorf("jumlah batch = %d, want 4 (%v)", len(rows), batchIDs(rows))
	}
}

func TestAdminBatches_UpcomingDropsDeparturesAlreadyGone(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)

	rows := h.as("admin").GET("/api/v1/admin/batches?upcoming=true").
		expectCode(http.StatusOK).batchesOf()

	for _, row := range rows {
		if row.ID == "batch-halal-lampau" {
			t.Fatalf("upcoming=true masih mengembalikan keberangkatan lampau: %v", batchIDs(rows))
		}
	}
	if len(rows) != 3 {
		t.Errorf("jumlah batch mendatang = %d, want 3 (%v)", len(rows), batchIDs(rows))
	}
}

// A tour leader opening the airport page wants the departure they are working
// today, not the one forty batches down the list.
func TestAdminBatches_NearestDepartureComesFirst(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)

	rows := h.as("admin").GET("/api/v1/admin/batches?upcoming=true").
		expectCode(http.StatusOK).batchesOf()

	want := []string{"batch-umroh-dekat", "batch-halal-dekat", "batch-umroh-jauh"}
	got := batchIDs(rows)
	if len(got) != len(want) {
		t.Fatalf("urutan = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("urutan = %v, want %v", got, want)
		}
	}
}

// A departure the operator has to recognise by sight: which trip it is, and how
// many people are already on it.
func TestAdminBatches_EachRowNamesItsPackageAndCountsItsParticipants(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)
	for i, batchID := range []string{"batch-umroh-dekat", "batch-umroh-dekat", "batch-halal-dekat"} {
		h.Participants.Seed(domainParticipant.Participant{
			ID: fmt.Sprintf("peserta-%d", i+1), BatchID: batchID,
			Name: fmt.Sprintf("Peserta %d", i+1), Phone: fmt.Sprintf("62811000000%d", i+1),
			IsActive: true,
		})
	}

	rows := h.as("admin").GET("/api/v1/admin/batches?upcoming=true").
		expectCode(http.StatusOK).batchesOf()

	byID := map[string]batchRow{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	if got := byID["batch-umroh-dekat"].PackageName; got != "Umroh Plus Istanbul" {
		t.Errorf("package_name = %q, want %q", got, "Umroh Plus Istanbul")
	}
	for batchID, want := range map[string]int{
		"batch-umroh-dekat": 2, "batch-halal-dekat": 1, "batch-umroh-jauh": 0,
	} {
		got := byID[batchID].ParticipantCount
		if got == nil {
			t.Errorf("participant_count %s tidak ada, want %d", batchID, want)
			continue
		}
		if *got != want {
			t.Errorf("participant_count %s = %d, want %d", batchID, *got, want)
		}
	}
	if got := byID["batch-umroh-dekat"].Quota; got != 40 {
		t.Errorf("quota = %d, want 40", got)
	}
	if byID["batch-umroh-dekat"].DepartureDate == "" {
		t.Error("departure_date kosong — pemilih tidak punya apa pun untuk ditampilkan")
	}
}

// All three statuses of the package_batches check constraint, so a picker can
// show a full or closed departure and say why it is unavailable.
func TestAdminBatches_FiltersByStatus(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)

	for status, want := range map[string][]string{
		"tersedia": {"batch-umroh-dekat", "batch-umroh-jauh"},
		"penuh":    {"batch-halal-dekat"},
		"ditutup":  {"batch-halal-lampau"},
	} {
		rows := h.as("admin").GET("/api/v1/admin/batches?status=" + status).
			expectCode(http.StatusOK).batchesOf()
		got := batchIDs(rows)
		if len(got) != len(want) {
			t.Errorf("status=%s mengembalikan %v, want %v", status, got, want)
			continue
		}
		found := map[string]bool{}
		for _, id := range got {
			found[id] = true
		}
		for _, id := range want {
			if !found[id] {
				t.Errorf("status=%s mengembalikan %v, want memuat %s", status, got, id)
			}
		}
	}
}

func TestAdminBatches_SearchMatchesPackageName(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)

	rows := h.as("admin").GET("/api/v1/admin/batches?search=istanbul").
		expectCode(http.StatusOK).batchesOf()

	if len(rows) != 2 {
		t.Fatalf("search=istanbul mengembalikan %v, want dua batch Umroh Plus Istanbul", batchIDs(rows))
	}
	for _, row := range rows {
		if row.PackageID != "package-umroh" {
			t.Errorf("search=istanbul ikut mengembalikan %s dari paket %s", row.ID, row.PackageName)
		}
	}
}

func TestAdminBatches_Paginates(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)

	res := h.as("admin").GET("/api/v1/admin/batches?page=1&per_page=2").expectCode(http.StatusOK)
	if rows := res.batchesOf(); len(rows) != 2 {
		t.Errorf("per_page=2 mengembalikan %d baris, want 2", len(rows))
	}
	if got := res.metaLimit(); got != 2 {
		t.Errorf("meta.limit = %d, want 2", got)
	}

	second := h.as("admin").GET("/api/v1/admin/batches?page=2&per_page=2").
		expectCode(http.StatusOK).batchesOf()
	if len(second) != 2 {
		t.Errorf("halaman kedua mengembalikan %d baris, want 2", len(second))
	}
}

func TestAdminBatches_RejectsRequestWithoutToken(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)

	h.anonymous().GET("/api/v1/admin/batches").expectCode(http.StatusUnauthorized)
}

// The airport page needs this read, so tour_leader is on the allow-list even
// though every other package route is ops-only. konsultan is not: nothing a
// konsultan does asks for a departure outside the lead they are converting.
// Setiap peran staf memilih keberangkatan di suatu tempat, jadi setiap peran
// staf boleh membacanya.
//
// Konsultan mula-mula tidak termasuk, dengan alasan satu-satunya tempat ia
// memilih batch adalah dialog konversi yang memakai daftar per-paket. Alasan itu
// melewatkan filter keberangkatan di halaman Peserta — halaman yang memang boleh
// dibuka konsultan — sehingga baginya filter itu hanya menampilkan "Gagal memuat
// daftar keberangkatan".
func TestAdminBatches_RoleMatrix(t *testing.T) {
	for _, role := range staffRoles {
		t.Run(role, func(t *testing.T) {
			h := newHarness(t)
			seedTwoPackagesOfBatches(h)

			h.as(role).GET("/api/v1/admin/batches").expectCode(http.StatusOK)
		})
	}
}

// Membuka daftar keberangkatan tidak boleh ikut membuka penanganan bandara.
// Keduanya sempat berbagi grup rute, sehingga memperluas yang satu berisiko
// memperluas yang lain tanpa ada yang memutuskannya.
func TestAdminBatches_DoesNotOpenAirportHandling(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)

	h.as("konsultan").GET("/api/v1/admin/batches").expectCode(http.StatusOK)
	h.as("konsultan").GET("/api/v1/admin/airport/checklist?batch_id=batch-1").
		expectCode(http.StatusForbidden)
}

// A head count belongs to the staff who manage the departure. The per-package
// listing is also what the public catalogue is served from, so counting there
// would have told every anonymous visitor how many seats each trip has sold.
func TestBatchListings_OnlyTheAdminListingCountsParticipants(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)
	h.Participants.Seed(domainParticipant.Participant{
		ID: "peserta-1", BatchID: "batch-umroh-dekat", Name: "Peserta Satu",
		Phone: "628110000001", IsActive: true,
	})

	var catalogue struct {
		Data struct {
			Batches []map[string]any `json:"batches"`
		} `json:"data"`
	}
	h.anonymous().GET("/api/v1/packages/umroh-plus-istanbul").
		expectCode(http.StatusOK).decode(&catalogue)
	if len(catalogue.Data.Batches) == 0 {
		t.Fatal("katalog publik tidak mengembalikan batch sama sekali")
	}
	for _, b := range catalogue.Data.Batches {
		if _, disclosed := b["participant_count"]; disclosed {
			t.Errorf("katalog publik menyiarkan participant_count: %v", b)
		}
	}

	for _, row := range h.as("admin").GET("/api/v1/admin/batches").
		expectCode(http.StatusOK).batchesOf() {
		if row.ParticipantCount == nil {
			t.Errorf("daftar admin tidak menghitung peserta untuk %s", row.ID)
		}
	}
}

// A konsultan converts their own leads, and the conversion dialog offers the
// departures of the lead's package. Listing those was ops-only, so the one role
// whose daily work needs it got a 403 and an empty picker.
func TestPackageBatchListing_KonsultanCanOfferDeparturesWhenConverting(t *testing.T) {
	h := newHarness(t)
	seedTwoPackagesOfBatches(h)

	h.as("konsultan").GET("/api/v1/admin/packages/package-umroh/batches").
		expectCode(http.StatusOK)
	// Creating departures is a different authority and stays where it was.
	h.as("konsultan").POST("/api/v1/admin/packages/package-umroh/batches", map[string]any{}).
		expectCode(http.StatusForbidden)
	h.as("tour_leader").GET("/api/v1/admin/packages/package-umroh/batches").
		expectCode(http.StatusForbidden)
}

// ─── Tiket 03: pencarian peserta ──────────────────────────────────────────────

// participantNames decodes the paginated participant list down to the names,
// which is all the picker assertions are about.
func (r *response) participantNames() []string {
	r.t.Helper()
	var envelope struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	r.decode(&envelope)
	names := make([]string, 0, len(envelope.Data))
	for _, row := range envelope.Data {
		names = append(names, row.Name)
	}
	return names
}

func seedParticipantsToSearch(h *harness) {
	h.Participants.Seed(domainParticipant.Participant{
		ID: "peserta-siti", BatchID: "batch-umroh-dekat", Name: "Siti Rahmawati",
		Phone: "628121234567", Email: "siti@pintour.test", IsActive: true,
	})
	h.Participants.Seed(domainParticipant.Participant{
		ID: "peserta-budi", BatchID: "batch-umroh-dekat", Name: "Budi Santoso",
		Phone: "628997654321", Email: "budi@pintour.test", IsActive: true,
	})
}

// An admin types what they remember — half a name, or the tail of a number.
func TestParticipantSearch_MatchesFragmentsOfNameAndPhone(t *testing.T) {
	h := newHarness(t)
	seedParticipantsToSearch(h)

	for _, tc := range []struct {
		query string
		want  string
	}{
		{query: "rahma", want: "Siti Rahmawati"},
		{query: "Budi", want: "Budi Santoso"},
		{query: "7654321", want: "Budi Santoso"},
		{query: "62812", want: "Siti Rahmawati"},
	} {
		names := h.as("admin").GET("/api/v1/admin/participants?search=" + tc.query).
			expectCode(http.StatusOK).participantNames()
		if len(names) != 1 || names[0] != tc.want {
			t.Errorf("search=%q mengembalikan %v, want [%s]", tc.query, names, tc.want)
		}
	}
}

// The search box must not become the way around the scoping that the participant
// list already applies — a konsultan searching still only reaches their own.
func TestParticipantSearch_KonsultanReachesOnlyTheirOwnParticipants(t *testing.T) {
	h := newHarness(t)
	seedParticipantsToSearch(h)

	mine, theirs := "user-konsultan", "konsultan-lain"
	h.Leads.Seed(domainLead.Lead{
		ID: "lead-siti", Name: "Siti Rahmawati", Phone: "628121234567",
		PackageID: "package-umroh", Status: "peserta", AssignedTo: &mine,
	})
	h.Leads.Seed(domainLead.Lead{
		ID: "lead-budi", Name: "Budi Santoso", Phone: "628997654321",
		PackageID: "package-umroh", Status: "peserta", AssignedTo: &theirs,
	})
	siti, budi := "lead-siti", "lead-budi"
	h.Participants.Seed(domainParticipant.Participant{
		ID: "peserta-siti", LeadID: &siti, BatchID: "batch-umroh-dekat",
		Name: "Siti Rahmawati", Phone: "628121234567", IsActive: true,
	})
	h.Participants.Seed(domainParticipant.Participant{
		ID: "peserta-budi", LeadID: &budi, BatchID: "batch-umroh-dekat",
		Name: "Budi Santoso", Phone: "628997654321", IsActive: true,
	})

	// An empty search already scopes; the point is that naming someone else does
	// not lift it.
	if names := h.as("konsultan").GET("/api/v1/admin/participants?search=Budi").
		expectCode(http.StatusOK).participantNames(); len(names) != 0 {
		t.Errorf("konsultan mencari peserta milik rekannya dan menerima %v, want kosong", names)
	}
	if names := h.as("konsultan").GET("/api/v1/admin/participants?search=Siti").
		expectCode(http.StatusOK).participantNames(); len(names) != 1 || names[0] != "Siti Rahmawati" {
		t.Errorf("konsultan mencari pesertanya sendiri dan menerima %v, want [Siti Rahmawati]", names)
	}
}
