// Package uow lets the application layer run several repository operations as
// one all-or-nothing unit.
//
// Some writes only make sense together. Converting a lead creates the portal
// identity a customer logs in with, the participant row that identity points at,
// and marks the lead converted; any two of those without the third is a state
// nobody asked for. Run separately, a failure halfway leaves the earlier rows
// behind, and the retry then reads them as history rather than as wreckage.
//
// What this package deliberately does not do is say how the guarantee is kept.
// The application layer receives a Runner and never learns whether it sits on a
// database transaction, so no storage type reaches it. It is offered alongside
// the plain repository interfaces rather than replacing them: an operation that
// writes one row keeps calling its repository directly.
package uow

import (
	"context"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
)

// Repos are the repositories handed to a unit of work, each bound to it. They
// are the ordinary domain interfaces — code inside a unit reads no differently
// from code outside one, it just uses the ones it was given here rather than the
// ones it holds.
//
// The set is the one the conversion flow needs. A later unit spanning different
// aggregates adds its repositories here; adding a field costs every implementor
// one line, which is the price of the application layer never seeing a
// transaction handle.
type Repos struct {
	Participants participant.Repository
	Leads        lead.Repository
	PortalUsers  portaluser.Repository
	// Added for the settlement unit: recording a gateway payment claims the
	// notification, writes the proof, and settles the invoice, and any of those
	// without the others is a money bug — a claimed notification whose proof was
	// never written loses a payment silently, which is exactly what a gateway
	// retry exists to prevent.
	Invoices      invoice.Repository
	Proofs        invoice.PaymentProofRepository
	GatewayOrders invoice.GatewayOrderRepository
}

// Runner runs work as a single unit.
type Runner interface {
	// Do runs fn with repositories bound to one unit of work. Every write fn
	// makes takes effect together if it returns nil, and none of them do if it
	// returns an error — which Do then returns unchanged, so callers can match
	// on their own sentinels through it.
	//
	// What binds a write to the unit is the repository it goes through, not the
	// context: fn must use the repositories it is handed, and calling one the
	// caller closed over writes outside the unit and will not be undone.
	Do(ctx context.Context, fn func(ctx context.Context, repos Repos) error) error
}
