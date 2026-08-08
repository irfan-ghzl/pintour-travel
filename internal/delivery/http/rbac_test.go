package httpdelivery

// Proof-of-concept coverage for the access matrix of PRD §5.3, one representative
// endpoint per route group. These exist to show the harness reaches every group
// and that RBAC is enforced server-side; the per-endpoint behaviour of each group
// belongs to the tickets that touch it.

import (
	"net/http"
	"testing"
)

// routeGroup is one row of the §5.3 matrix: an endpoint plus the roles allowed.
type routeGroup struct {
	name    string
	method  string
	path    string
	body    any
	allowed []string
}

func (g routeGroup) permits(role string) bool {
	for _, r := range g.allowed {
		if r == role {
			return true
		}
	}
	return false
}

// routeGroups covers each group RegisterRoutes creates, with an endpoint that
// only reads so the assertion is about access, not about side effects.
var routeGroups = []routeGroup{
	{
		name:    "base staff (own profile)",
		method:  http.MethodGet,
		path:    "/api/v1/admin/auth/me",
		allowed: []string{"super_admin", "admin", "konsultan", "tour_leader"},
	},
	{
		name:    "ops (invoices)",
		method:  http.MethodGet,
		path:    "/api/v1/admin/invoices",
		allowed: []string{"super_admin", "admin"},
	},
	{
		name:    "sales (CRM leads)",
		method:  http.MethodGet,
		path:    "/api/v1/admin/leads",
		allowed: []string{"super_admin", "admin", "konsultan"},
	},
	{
		name:    "airport handling",
		method:  http.MethodGet,
		path:    "/api/v1/admin/airport/checklist?batch_id=batch-1",
		allowed: []string{"super_admin", "admin", "tour_leader"},
	},
	{
		name:    "super admin (user management)",
		method:  http.MethodGet,
		path:    "/api/v1/admin/users",
		allowed: []string{"super_admin"},
	},
}

func TestRouteGroups_EnforceRoleMatrix(t *testing.T) {
	for _, group := range routeGroups {
		for _, role := range staffRoles {
			t.Run(group.name+"/"+role, func(t *testing.T) {
				h := newHarness(t)
				h.seedBaseline()

				res := h.as(role).do(group.method, group.path, group.body)
				if group.permits(role) {
					res.expectCode(http.StatusOK)
				} else {
					res.expectCode(http.StatusForbidden)
				}
			})
		}
	}
}

func TestStaffRoutes_RejectAnonymousRequests(t *testing.T) {
	for _, group := range routeGroups {
		t.Run(group.name, func(t *testing.T) {
			h := newHarness(t)
			h.anonymous().do(group.method, group.path, group.body).
				expectCode(http.StatusUnauthorized)
		})
	}
}

// A portal token is signed with the same secret as a staff token but carries
// only participant_id. §19.1 requires it to be refused everywhere under /admin.
func TestStaffRoutes_RejectPortalToken(t *testing.T) {
	for _, group := range routeGroups {
		t.Run(group.name, func(t *testing.T) {
			h := newHarness(t)
			h.seedBaseline()

			portalAsStaff := h.anonymous().withHeader(
				"Authorization", "Bearer "+h.portalToken("participant-1", "portal-user-1"),
			)
			portalAsStaff.do(group.method, group.path, group.body).
				expectCode(http.StatusUnauthorized)
		})
	}
}

// The portal group is reachable with a portal token and refuses requests without
// one — the other half of the §19.1 separation asserted above.
func TestPortalRoutes_RequirePortalToken(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	h.asParticipant("participant-1").GET("/api/v1/portal/me").expectCode(http.StatusOK)
	h.anonymous().GET("/api/v1/portal/me").expectCode(http.StatusUnauthorized)
}

// FR-RPT-02 opens analytics to every staff role, and the base group carries no
// RequireRole, so all four reach it.
//
// This is also the known gap documented in harness_test.go: the handler reads
// Postgres directly, so nothing about its body can be asserted here. What can be
// asserted is that the request got past the middleware — anything other than 403
// proves the route is not role-gated. The exact status is deliberately not
// pinned: today it is 200 with zeroed figures because the handler discards its
// query errors, and a ticket that stops it swallowing them should not break an
// access-control test.
func TestDashboardAnalytics_NotRoleGated(t *testing.T) {
	for _, role := range staffRoles {
		t.Run(role, func(t *testing.T) {
			h := newHarness(t)
			res := h.as(role).GET("/api/v1/admin/dashboard/analytics")
			if res.Code == http.StatusForbidden {
				t.Errorf("role %q got 403; FR-RPT-02 opens analytics to every staff role", role)
			}
		})
	}
}
