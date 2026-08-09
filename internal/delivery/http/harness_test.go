package httpdelivery

// HTTP test harness.
//
// newHarness builds a ready-to-serve Echo instance from the fake repositories in
// fakes_test.go and the REAL application services, wired through the same
// RegisterRoutes the production binary calls. A request made through this
// harness therefore travels the whole path — routing, middleware, handler,
// application service, domain — which is why behaviour promised by the PRD can
// be asserted here rather than at the service layer.
//
// Typical use:
//
//	h := newHarness(t)
//	res := h.as("konsultan").GET("/api/v1/admin/invoices")
//	res.expectCode(http.StatusForbidden)
//
// Nothing here touches Docker, Postgres, or the network. Every external adapter
// (WhatsApp, email, storage, payment gateway, chatbot, OCR) is constructed with
// empty credentials, which puts it in the no-op mode it already has in
// production.
//
// ─── Known gaps ───────────────────────────────────────────────────────────────
//
//  1. Dashboard/analytics (GET /api/v1/admin/dashboard/*) and reports
//     (GET /api/v1/admin/reports/export) take a *sql.DB and run SQL directly
//     instead of going through a repository interface, so their responses cannot
//     be faked. The harness gives them a *sql.DB whose connections always fail:
//     enough to prove the routes are reachable and correctly role-gated (the
//     middleware answers before the query runs), not enough to assert what they
//     return. Analytics in particular discards its query errors, so it answers
//     200 with zeroed figures here — the status is meaningful, the body is not.
//     Testing their output needs a repository seam for these handlers, or the
//     Postgres contract seam described in the spec as "seam kedua".
//
//  2. Outbound calls to the payment gateway are only testable because
//     service.WithMidtransBaseURL lets a test point the client at a local
//     httptest server (see withMidtransServer). Every other external adapter is
//     exercised only in its no-op mode; asserting what they would have sent
//     needs its own seam.
//
//  3. CLOSED by tiket 08. WhatsApp sends were unobservable — FonnteService.Send
//     returned at "apiToken == ''" BEFORE writing its notification row, and its
//     endpoint was hardcoded, so h.Notifications stayed empty whatever a flow
//     sent. service.WithFonnteBaseURL now lets withFonnteServer point the client
//     at a local stand-in with a fake token, and Send runs the whole way through
//     without touching the network. Tests that do not pass the option still get
//     the tokenless no-op, which is what a deployment without a gateway does.
//
// One caveat rather than a gap: EmailService.Send with an empty key returns an
// error instead of no-opping like the others. Every current caller discards it
// (mostly in a goroutine), so no route fails because of it — but a route that
// starts propagating that error would behave differently here than in a
// deployment with Resend configured.

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	leadsvc "github.com/irfan-ghzl/pintour-travel/internal/application/lead"
	pkgsvc "github.com/irfan-ghzl/pintour-travel/internal/application/package"
	participantsvc "github.com/irfan-ghzl/pintour-travel/internal/application/participant"
	usersvc "github.com/irfan-ghzl/pintour-travel/internal/application/user"
	"github.com/irfan-ghzl/pintour-travel/internal/auth"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
	"github.com/labstack/echo/v4"
)

// staffRoles lists the four staff roles of PRD §5.3, in privilege order. Table
// tests over the RBAC matrix iterate this.
var staffRoles = []string{"super_admin", "admin", "konsultan", "tour_leader"}

const testJWTSecret = "harness-secret"

// harness is a running server plus direct access to the state behind it.
type harness struct {
	t   *testing.T
	e   *echo.Echo
	svc Services

	// Fakes, exposed so a test can seed what a request should see and read back
	// what it wrote.
	Users         *fakeUserRepo
	TourLeaders   *fakeTourLeaderRepo
	Leads         *fakeLeadRepo
	Notes         *fakeNoteRepo
	Participants  *fakeParticipantRepo
	PortalUsers   *fakePortalUserRepo
	Invoices      *fakeInvoiceRepo
	Proofs        *fakeProofRepo
	Packages      *fakePackageRepo
	Images        *fakeImageRepo
	Batches       *fakeBatchRepo
	Documents     *fakeDocumentRepo
	OCRResults    *fakeOCRResultRepo
	CountryReqs   *fakeCountryReqRepo
	Airport       *fakeAirportRepo
	Notifications *fakeNotifRepo
	Chatbot       *fakeChatbotRepo
	GatewayOrders *fakeGatewayOrderRepo

	// Unit runs a unit of work over the fakes above, undoing its writes when it
	// fails — see fakeUnitOfWork for why a passthrough would not do.
	Unit *fakeUnitOfWork
}

// harnessOption customizes the harness before routes are registered.
type harnessOption func(*harnessConfig)

type harnessConfig struct {
	midtransServerKey string
	midtransClientKey string
	midtransBaseURL   string
	chatbotToken      string
	storageBaseURL    string
	storageServiceKey string
	fonnteBaseURL     string
	fonnteToken       string
}

// withMidtransServer enables the payment gateway and points it at baseURL —
// normally an httptest.Server standing in for Snap. Without this option the
// gateway stays unconfigured and refuses to create transactions, as it does in a
// deployment with no keys.
func withMidtransServer(baseURL string) harnessOption {
	return func(c *harnessConfig) {
		c.midtransServerKey = "SB-Mid-server-test"
		c.midtransClientKey = "SB-Mid-client-test"
		c.midtransBaseURL = baseURL
	}
}

// withChatbotToken sets the shared token the Fonnte inbound webhook checks.
func withChatbotToken(token string) harnessOption {
	return func(c *harnessConfig) { c.chatbotToken = token }
}

// withStorageServer enables object storage and points it at baseURL — normally
// an httptest.Server standing in for Supabase Storage. Without this option the
// adapter stays unconfigured and refuses to sign, as it does in a deployment
// with no keys. Unlike the payment gateway this needed no production change:
// NewStorageService already takes its base URL from configuration.
func withStorageServer(baseURL string) harnessOption {
	return func(c *harnessConfig) {
		c.storageBaseURL = baseURL
		c.storageServiceKey = "storage-key-test"
	}
}

// withFonnteServer enables the WhatsApp gateway and points it at baseURL —
// normally an httptest.Server standing in for Fonnte. Without it the adapter
// stays tokenless and Send returns before writing anything, which is why
// h.Notifications used to be empty no matter what a flow sent (gap 3 above).
// With it, Send runs the whole way through — writes the row, posts, updates the
// status — and never leaves the machine.
func withFonnteServer(baseURL string) harnessOption {
	return func(c *harnessConfig) {
		c.fonnteBaseURL = baseURL
		c.fonnteToken = "fonnte-token-test"
	}
}

// newHarness builds a server backed by fresh fakes. It seeds one staff user per
// role (id "user-<role>") so tokens issued by as() resolve to a real account.
func newHarness(t *testing.T, opts ...harnessOption) *harness {
	t.Helper()

	cfg := harnessConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	h := &harness{
		t:             t,
		e:             echo.New(),
		Users:         newFakeUserRepo(),
		TourLeaders:   newFakeTourLeaderRepo(),
		Leads:         newFakeLeadRepo(),
		Notes:         newFakeNoteRepo(),
		Participants:  newFakeParticipantRepo(),
		PortalUsers:   newFakePortalUserRepo(),
		Invoices:      newFakeInvoiceRepo(),
		Proofs:        newFakeProofRepo(),
		Packages:      newFakePackageRepo(),
		Images:        newFakeImageRepo(),
		Batches:       newFakeBatchRepo(),
		Documents:     newFakeDocumentRepo(),
		OCRResults:    newFakeOCRResultRepo(),
		CountryReqs:   newFakeCountryReqRepo(),
		Airport:       newFakeAirportRepo(),
		Notifications: newFakeNotifRepo(),
		Chatbot:       newFakeChatbotRepo(),
		GatewayOrders: newFakeGatewayOrderRepo(),
	}
	h.Unit = &fakeUnitOfWork{
		participants: h.Participants, leads: h.Leads, portalUsers: h.PortalUsers,
		invoices: h.Invoices, proofs: h.Proofs, gatewayOrders: h.GatewayOrders,
	}
	h.e.HideBanner = true
	h.e.Logger.SetOutput(io.Discard)

	for _, role := range staffRoles {
		h.Users.Seed(domainUser.User{
			ID:       "user-" + role,
			Name:     "Staff " + role,
			Email:    role + "@pintour.test",
			Role:     role,
			IsActive: true,
		})
	}

	// External adapters: empty credentials put each one in the no-op mode it
	// already has in production, so nothing reaches the network. The two that a
	// test may point at a local httptest server (payment gateway, storage) stay
	// unconfigured unless the corresponding option was passed.
	fonnte := service.NewFonnteService(cfg.fonnteToken, h.Notifications,
		service.WithFonnteBaseURL(cfg.fonnteBaseURL))
	email := service.NewEmailService("", "")
	storage := service.NewStorageService(cfg.storageBaseURL, cfg.storageServiceKey)
	pdf := service.NewPDFService()
	midtrans := service.NewMidtransService(
		cfg.midtransServerKey, cfg.midtransClientKey, "sandbox",
		service.WithMidtransBaseURL(cfg.midtransBaseURL),
	)
	ocr := service.NewOCRService("none", "", "", 0, h.OCRResults, h.Participants, storage)

	// See "Known gaps" above: nothing can be seeded behind this handle, only the
	// failure it produces.
	db := unreachableDB()

	chatbot := service.NewChatbotService(h.Chatbot, db, h.Packages, fonnte, "", "", 0, false)

	invoiceSvc := invoicesvc.NewService(h.Invoices, h.Proofs, h.Participants, h.GatewayOrders,
		h.Unit, fonnte, pdf, email, midtrans)
	leadSvc := leadsvc.NewService(h.Leads, h.Notes, h.Users, h.PortalUsers, h.Participants, fonnte, email)
	participantSvc := participantsvc.NewService(h.Participants, h.Leads, h.PortalUsers,
		h.Unit, h.Batches, h.Packages, invoiceSvc, h.CountryReqs, fonnte)
	packageSvc := pkgsvc.NewService(h.Packages, h.Images, h.Batches)
	userSvc := usersvc.NewUserService(h.Users, testJWTSecret, 24)

	h.svc = Services{
		Package:        packageSvc,
		Lead:           leadSvc,
		Participant:    participantSvc,
		Invoice:        invoiceSvc,
		User:           userSvc,
		UserRepo:       h.Users,
		TourLeaderRepo: h.TourLeaders,
		Airport:        h.Airport,
		Participants:   h.Participants,
		Document:       h.Documents,
		OCRRepo:        h.OCRResults,
		OCR:            ocr,
		CountryReq:     h.CountryReqs,
		PDF:            pdf,
		Fonnte:         fonnte,
		Email:          email,
		Storage:        storage,
		NotifRepo:      h.Notifications,
		DB:             db,
		Midtrans:       midtrans,
		Chatbot:        chatbot,
		ChatbotRepo:    h.Chatbot,
		ChatbotToken:   cfg.chatbotToken,
		AppURL:         "http://harness.test",
		PortalURL:      "http://portal.harness.test",
		JWTSecret:      testJWTSecret,
		JWTExpiryHours: 24,
		// Only affects the Secure flag on the session cookie; tests speak plain
		// HTTP, so they run as a non-production deployment does.
		Production: false,
	}

	RegisterRoutes(h.e, h.svc)
	t.Cleanup(func() { _ = db.Close() })
	return h
}

// ─── Seeding ──────────────────────────────────────────────────────────────────

// seedBaseline fills the fakes with one row per aggregate the routes read: a
// package, a lead, a participant, and an unpaid invoice belonging to that
// participant. It exists so a test asserting access control does not have to
// restate the world; a test that cares about a particular shape seeds its own
// rows over the top, or skips this entirely.
func (h *harness) seedBaseline() {
	h.Packages.Seed(domainPkg.Package{
		ID: "package-1", Name: "Umroh Reguler 9 Hari", Slug: "umroh-reguler-9-hari",
		Destination: "Arab Saudi", Category: "umroh", DurationDays: 9,
		BasePrice: 25000000, IsActive: true,
	})
	h.Leads.Seed(domainLead.Lead{
		ID: "lead-1", Name: "Prospek Satu", Phone: "628111111111",
		PackageID: "package-1", Status: "baru", CreatedAt: time.Now(),
	})
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-1", BatchID: "batch-1", Name: "Peserta Satu",
		Phone: "628222222222", Email: "peserta@pintour.test",
		PackageName: "Umroh Reguler 9 Hari", RoomType: "double", IsActive: true,
	})
	h.Invoices.Seed(domainInvoice.Invoice{
		ID: "invoice-1", InvoiceNumber: "INV-202608-0001",
		ParticipantID: "participant-1", BatchID: "batch-1",
		Amount: 25000000, Status: "diterbitkan", DueDate: calendar.Today().AddDays(1),
	})
}

// ─── Identity ─────────────────────────────────────────────────────────────────

// as returns a client that signs every request as the seeded staff user for
// role — "as an admin", in one line.
func (h *harness) as(role string) *client {
	return h.asUser("user-"+role, role)
}

// asUser signs as a specific staff account, for handlers that scope results to
// the caller (a konsultan seeing only their own leads).
func (h *harness) asUser(userID, role string) *client {
	return &client{h: h, headers: map[string]string{
		"Authorization": "Bearer " + h.staffToken(userID, role),
	}}
}

// asParticipant signs with a portal token for the given participant. Pass a
// portal-user id too for a returning customer whose account spans several tours.
func (h *harness) asParticipant(participantID string, portalUserID ...string) *client {
	portalUser := ""
	if len(portalUserID) > 0 {
		portalUser = portalUserID[0]
	}
	return &client{h: h, headers: map[string]string{
		"X-Portal-Token": h.portalToken(participantID, portalUser),
	}}
}

// anonymous returns a client that sends no credentials at all.
func (h *harness) anonymous() *client { return &client{h: h, headers: map[string]string{}} }

// staffToken mints a staff JWT the same shape UserService.Login issues.
func (h *harness) staffToken(userID, role string) string {
	return h.sign(&auth.Claims{UserID: userID, Role: role, RegisteredClaims: shortLivedClaims()})
}

// portalToken mints a participant portal JWT, the same shape PortalLogin issues.
func (h *harness) portalToken(participantID, portalUserID string) string {
	return h.sign(&portalClaims{
		ParticipantID:    participantID,
		PortalUserID:     portalUserID,
		RegisteredClaims: shortLivedClaims(),
	})
}

// sign signs claims with the harness secret — the same secret RegisterRoutes was
// given, so both middlewares accept what comes out.
func (h *harness) sign(claims jwt.Claims) string {
	h.t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	if err != nil {
		h.t.Fatalf("sign token: %v", err)
	}
	return token
}

// shortLivedClaims returns registered claims that outlast a test run but little
// else, so an accidentally leaked harness token is worthless.
func shortLivedClaims() jwt.RegisteredClaims {
	now := time.Now()
	return jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
}

// ─── Requests ─────────────────────────────────────────────────────────────────

// client issues requests carrying one identity's credentials.
type client struct {
	h       *harness
	headers map[string]string
}

// withHeader returns a copy of the client that also sends the given header.
func (c *client) withHeader(name, value string) *client {
	headers := make(map[string]string, len(c.headers)+1)
	for k, v := range c.headers {
		headers[k] = v
	}
	headers[name] = value
	return &client{h: c.h, headers: headers}
}

// One method per verb the router serves. PUT and DELETE have no caller yet —
// they are here because a request client covering only some of a REST API's
// verbs is an unfinished client, and this ticket's deliverable is the client.
func (c *client) GET(path string) *response    { return c.do(http.MethodGet, path, nil) }
func (c *client) DELETE(path string) *response { return c.do(http.MethodDelete, path, nil) }

func (c *client) POST(path string, body any) *response {
	return c.do(http.MethodPost, path, body)
}
func (c *client) PUT(path string, body any) *response { return c.do(http.MethodPut, path, body) }
func (c *client) PATCH(path string, body any) *response {
	return c.do(http.MethodPatch, path, body)
}

// do serves one request through the full Echo stack. body is JSON-encoded unless
// it is nil, a string, or []byte, which are sent verbatim (for malformed-payload
// and body-size tests).
func (c *client) do(method, path string, body any) *response {
	c.h.t.Helper()

	var reader io.Reader
	contentType := ""
	switch v := body.(type) {
	case nil:
	case string:
		reader = bytes.NewBufferString(v)
		contentType = echo.MIMEApplicationJSON
	case []byte:
		reader = bytes.NewBuffer(v)
		contentType = echo.MIMEApplicationJSON
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			c.h.t.Fatalf("encode request body: %v", err)
		}
		reader = bytes.NewBuffer(encoded)
		contentType = echo.MIMEApplicationJSON
	}

	req := httptest.NewRequest(method, path, reader)
	if contentType != "" {
		req.Header.Set(echo.HeaderContentType, contentType)
	}
	for name, value := range c.headers {
		req.Header.Set(name, value)
	}

	rec := httptest.NewRecorder()
	c.h.e.ServeHTTP(rec, req)
	return &response{
		t: c.h.t, method: method, path: path,
		Code: rec.Code, Body: rec.Body.Bytes(), Headers: rec.Header(),
	}
}

// response is one served request's outcome.
type response struct {
	t      *testing.T
	method string
	path   string
	Code   int
	Body   []byte
	// Headers is what the handler set on the way out — the content type of a
	// download, and the Set-Cookie a session depends on.
	Headers http.Header
}

// cookie returns the cookie the response set under name, or nil.
func (r *response) cookie(name string) *http.Cookie {
	r.t.Helper()
	for _, raw := range r.Headers.Values("Set-Cookie") {
		parsed, err := http.ParseSetCookie(raw)
		if err == nil && parsed.Name == name {
			return parsed
		}
	}
	return nil
}

// expectPDF fails the test unless the body is a PDF served as one.
func (r *response) expectPDF() *response {
	r.t.Helper()
	if got := r.Headers.Get("Content-Type"); got != "application/pdf" {
		r.t.Errorf("%s %s: Content-Type = %q, want application/pdf", r.method, r.path, got)
	}
	if len(r.Body) < 5 || string(r.Body[:5]) != "%PDF-" {
		r.t.Errorf("%s %s: badan respons bukan PDF", r.method, r.path)
	}
	return r
}

// expectCode fails the test unless the status matches.
func (r *response) expectCode(want int) *response {
	r.t.Helper()
	if r.Code != want {
		r.t.Errorf("%s %s: status = %d, want %d (body: %s)", r.method, r.path, r.Code, want, r.Body)
	}
	return r
}

// decode unmarshals the JSON body into dest.
func (r *response) decode(dest any) {
	r.t.Helper()
	if err := json.Unmarshal(r.Body, dest); err != nil {
		r.t.Fatalf("%s %s: decode body: %v (body: %s)", r.method, r.path, err, r.Body)
	}
}

// message returns the human-readable message of an error envelope.
func (r *response) message() string {
	r.t.Helper()
	var envelope struct {
		Message string `json:"message"`
	}
	r.decode(&envelope)
	return envelope.Message
}

// expectMessageMentions fails the test unless the error message names field —
// how a validation 400 is told apart from a generic one.
func (r *response) expectMessageMentions(field string) *response {
	r.t.Helper()
	if msg := r.message(); !strings.Contains(msg, field) {
		r.t.Errorf("%s %s: pesan %q tidak menyebut %q", r.method, r.path, msg, field)
	}
	return r
}

// metaLimit returns the page size the server reported in the pagination meta.
func (r *response) metaLimit() int {
	r.t.Helper()
	var envelope struct {
		Meta struct {
			Limit int `json:"limit"`
		} `json:"meta"`
	}
	r.decode(&envelope)
	return envelope.Meta.Limit
}

// data decodes the standard {"success":…,"data":…} envelope and returns data.
func (r *response) data() map[string]any {
	r.t.Helper()
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	r.decode(&envelope)
	return envelope.Data
}

// ─── Unreachable database ─────────────────────────────────────────────────────

// unreachableDB returns a *sql.DB that hands out no connections. Handlers that
// bypass the repository interfaces (see "Known gaps") then fail with an error
// instead of dereferencing nil, so the middleware in front of them can still be
// asserted.
func unreachableDB() *sql.DB { return sql.OpenDB(unreachableConnector{}) }

var errNoDatabase = errors.New("harness: no database (handler bypasses the repository seam)")

type unreachableConnector struct{}

func (unreachableConnector) Connect(context.Context) (driver.Conn, error) { return nil, errNoDatabase }
func (unreachableConnector) Driver() driver.Driver                        { return unreachableDriver{} }

type unreachableDriver struct{}

func (unreachableDriver) Open(string) (driver.Conn, error) { return nil, errNoDatabase }
