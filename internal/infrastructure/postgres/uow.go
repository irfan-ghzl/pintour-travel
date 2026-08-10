package postgres

import (
	"context"
	"database/sql"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/uow"
)

// dbtx is the query surface *sql.DB and *sql.Tx have in common. Repositories
// hold one of these rather than a *sql.DB, so the same repository code serves a
// plain connection and a transaction — a repository never learns which it is on,
// and none of them needed rewriting to gain the choice.
type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// unitOfWork runs a unit of work as one database transaction.
type unitOfWork struct{ db *sql.DB }

// NewUnitOfWork returns the Postgres unit-of-work runner. It sits alongside the
// NewXxxRepo constructors rather than replacing them: a caller that writes one
// row still takes a repository built straight from the connection.
func NewUnitOfWork(db *sql.DB) uow.Runner { return &unitOfWork{db} }

// Do opens a transaction, hands fn repositories bound to it, and commits only if
// fn succeeds.
//
// The deferred rollback covers every path that does not reach Commit, a panic
// from inside fn included; after a successful commit it is a no-op that reports
// sql.ErrTxDone, which is why its error is discarded. Leaving a transaction open
// is the one outcome worse than the half-written state this exists to prevent —
// it holds locks until the connection dies.
func (u *unitOfWork) Do(ctx context.Context, fn func(context.Context, uow.Repos) error) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(ctx, uow.Repos{
		Participants:  &participantRepo{tx},
		Leads:         &leadRepo{tx},
		PortalUsers:   &portalUserRepo{tx},
		Invoices:      &invoiceRepo{tx},
		Proofs:        &paymentProofRepo{tx},
		GatewayOrders: &gatewayOrderRepo{tx},
	}); err != nil {
		// Returned unchanged so the caller can still match its own sentinels.
		return err
	}
	return tx.Commit()
}
