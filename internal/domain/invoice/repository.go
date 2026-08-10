package invoice

import "context"

type Repository interface {
	Create(ctx context.Context, inv *Invoice) error
	Update(ctx context.Context, inv *Invoice) error
	GetByID(ctx context.Context, id string) (*Invoice, error)
	List(ctx context.Context, f Filter) ([]Invoice, int, error)
	Confirm(ctx context.Context, id, confirmedBy string) error
	NextSequence(ctx context.Context, yearMonth string) (int, error)
	// Midtrans (v2.0 F1)
	SetSnap(ctx context.Context, id, snapToken, orderID string) error
}

type PaymentProofRepository interface {
	Create(ctx context.Context, pp *PaymentProof) error
	GetByID(ctx context.Context, id string) (*PaymentProof, error)
	GetByInvoice(ctx context.Context, invoiceID string) ([]PaymentProof, error)
	Review(ctx context.Context, id, status, reviewedBy, notes string) error
}
