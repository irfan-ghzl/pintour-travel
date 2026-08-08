package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// This file implements the automation scheduler jobs from the prompt (§2).
// All jobs follow the existing pattern: raw SQL on s.db, log-and-continue on
// error (never panic), 1s spacing between WA sends (§17.2), and idempotency via
// wa_notifications so re-runs do not duplicate sends.

// checkStaleLeads (§2.1, hourly): remind the assigned consultant about leads
// still 'baru' after 24h. Deduped against LEADS_STALE sent in the last 24h.
func (s *Scheduler) checkStaleLeads() {
	if s.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT l.id, l.name, COALESCE(u.phone,''), COALESCE(u.name,'')
		FROM leads l
		JOIN users u ON u.id = l.assigned_to
		WHERE l.status='baru'
		  AND l.created_at <= NOW() - INTERVAL '24 hours'
		  AND l.deleted_at IS NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM wa_notifications w
		    WHERE w.reference_id = l.id
		      AND w.message_type = 'LEADS_STALE'
		      AND w.created_at > NOW() - INTERVAL '24 hours'
		  )`)
	if err != nil {
		log.Printf("checkStaleLeads query: %v", err)
		return
	}
	type stale struct{ id, name, phone, consultant string }
	var list []stale
	for rows.Next() {
		var st stale
		if err := rows.Scan(&st.id, &st.name, &st.phone, &st.consultant); err != nil {
			continue
		}
		list = append(list, st)
	}
	rows.Close()

	for _, st := range list {
		if st.phone == "" {
			continue
		}
		_ = s.fonnte.SendLeadsStale(ctx, st.phone, st.consultant, st.name, st.id)
		time.Sleep(time.Second)
	}
	if len(list) > 0 {
		log.Printf("checkStaleLeads: %d reminder terkirim", len(list))
	}
}

// expireLeads (§2.2, daily): mark leads inactive for 30 days as tidak_deal.
// The schema has no 'expired' status, so tidak_deal is the code-wins equivalent.
func (s *Scheduler) expireLeads() {
	if s.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// A bulk update still owes the audit trail a row per lead (FR-CRM-02) — a
	// nightly job quietly moving leads is exactly the hole the trail exists to
	// close. `candidates` is read from the pre-update snapshot so it still holds
	// the status each lead is leaving, which RETURNING cannot give us. changed_by
	// is NULL: nobody did this, the scheduler did.
	res, err := s.db.ExecContext(ctx, `
		WITH candidates AS (
			SELECT id, status FROM leads
			WHERE status NOT IN ('deal','tidak_deal','peserta')
			  AND updated_at <= NOW() - INTERVAL '30 days'
			  AND deleted_at IS NULL
		), updated AS (
			UPDATE leads SET status='tidak_deal', updated_at=NOW()
			WHERE id IN (SELECT id FROM candidates)
			RETURNING id
		)
		INSERT INTO lead_status_history (lead_id, from_status, to_status, changed_by)
		SELECT c.id, c.status, 'tidak_deal', NULL FROM candidates c`)
	if err != nil {
		log.Printf("expireLeads: %v", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("expireLeads: %d leads di-expire (→ tidak_deal)", n)
	}
}

// expireInvoices (§2.3, daily): notify participants whose invoice is past due
// (PAYMENT_OVERDUE WA) and send admins one digest email. No status change — the
// schema has no 'expired' invoice status (code-wins). Deduped per 24h.
func (s *Scheduler) expireInvoices() {
	if s.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT i.id, i.invoice_number, i.amount, p.name, COALESCE(p.phone,''), COALESCE(p.email,'')
		FROM invoices i
		JOIN participants p ON p.id = i.participant_id
		WHERE i.deleted_at IS NULL
		  AND i.status IN ('diterbitkan','menunggu_bayar')
		  AND i.due_date < NOW()
		  AND NOT EXISTS (
		    SELECT 1 FROM wa_notifications w
		    WHERE w.reference_id = i.id
		      AND w.message_type = 'PAYMENT_OVERDUE'
		      AND w.created_at > NOW() - INTERVAL '24 hours'
		  )
		ORDER BY i.due_date`)
	if err != nil {
		log.Printf("expireInvoices query: %v", err)
		return
	}
	type overdue struct {
		id, number, name, phone, email string
		amount                         float64
	}
	var list []overdue
	for rows.Next() {
		var o overdue
		if err := rows.Scan(&o.id, &o.number, &o.amount, &o.name, &o.phone, &o.email); err != nil {
			continue
		}
		list = append(list, o)
	}
	rows.Close()

	if len(list) == 0 {
		return
	}

	adminContact := envOr("ADMIN_CONTACT", "Tim Pintour")
	digest := make([]service.OverdueInvoiceRow, 0, len(list))
	for _, o := range list {
		if o.phone != "" {
			// Writes the PAYMENT_OVERDUE row this job dedupes on.
			_ = s.fonnte.SendPaymentOverdue(ctx, o.phone, o.name, o.number, adminContact, o.id)
			time.Sleep(time.Second)
		} else {
			// No phone means no WhatsApp send, and the marker row came from the
			// send — so the dedup found nothing and a participant without a
			// number was emailed the same overdue notice every single day. The
			// marker is recorded either way, following the pattern the quota
			// warning already uses.
			s.markNotified(ctx, o.id, notification.TypePaymentOverdue, o.name)
		}
		if s.email != nil && o.email != "" {
			_ = s.email.SendEmailPaymentOverdue(ctx, o.email, o.name, o.number, adminContact)
		}
		digest = append(digest, service.OverdueInvoiceRow{
			ParticipantName: o.name,
			InvoiceNumber:   o.number,
			Amount:          rupiah(o.amount),
		})
	}

	// One digest email to all admins.
	if s.email != nil {
		actionLink := envOr("APP_URL", "http://localhost:5173") + "/admin/invoices"
		for _, a := range s.getAdmins(ctx) {
			if a.Email == "" {
				continue
			}
			_ = s.email.SendEmailAdminPaymentOverdue(ctx, a.Email, digest, actionLink)
		}
	}
	log.Printf("expireInvoices: %d invoice overdue diproses", len(list))
}

// checkBatchQuota (§2.4, daily 08:00): warn admins when an open batch is ≤20%
// seats remaining. Deduped per batch per day via a QUOTA_WARNING marker row.
func (s *Scheduler) checkBatchQuota() {
	if s.db == nil || s.email == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT pb.id, pkg.name, pb.departure_date, pb.quota,
		  COALESCE((SELECT COUNT(*) FROM leads l WHERE l.batch_id = pb.id AND l.status='deal'),0) AS sold
		FROM package_batches pb
		JOIN packages pkg ON pkg.id = pb.package_id
		WHERE pb.status='tersedia'`)
	if err != nil {
		log.Printf("checkBatchQuota query: %v", err)
		return
	}
	type batch struct {
		id, name  string
		departure time.Time
		quota     int
		sold      int
	}
	var list []batch
	for rows.Next() {
		var b batch
		if err := rows.Scan(&b.id, &b.name, &b.departure, &b.quota, &b.sold); err != nil {
			continue
		}
		list = append(list, b)
	}
	rows.Close()

	admins := s.getAdmins(ctx)
	for _, b := range list {
		if b.quota <= 0 {
			continue
		}
		remaining := b.quota - b.sold
		if float64(remaining)/float64(b.quota)*100 > 20 {
			continue
		}
		// Dedupe per batch per day.
		var exists bool
		_ = s.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM wa_notifications
			WHERE reference_id=$1 AND message_type='QUOTA_WARNING'
			AND created_at::date = CURRENT_DATE)`, b.id).Scan(&exists)
		if exists {
			continue
		}

		depDate := b.departure.Format("02 Jan 2006")
		for _, a := range admins {
			if a.Email == "" {
				continue
			}
			_ = s.email.SendEmailAdminQuotaWarning(ctx, a.Email, b.name, depDate, remaining, b.quota)
		}
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO wa_notifications
			(id,recipient_phone,recipient_name,message_type,message_content,reference_id,reference_type,status,created_at)
			VALUES (gen_random_uuid(),'-','admin','QUOTA_WARNING',$1,$2,'batch','sent',NOW())`,
			fmt.Sprintf("Quota warning %s: sisa %d/%d", b.name, remaining, b.quota), b.id)
		log.Printf("checkBatchQuota: warning terkirim untuk batch %s (sisa %d/%d)", b.name, remaining, b.quota)
	}
}

// RunJobNow runs a single automation job once, on demand (dev/testing helper —
// the jobs are otherwise only triggered by cron). Safe to call without Start().
func (s *Scheduler) RunJobNow(name string) error {
	switch name {
	case "stale-leads":
		s.checkStaleLeads()
	case "expire-leads":
		s.expireLeads()
	case "expire-invoices":
		s.expireInvoices()
	case "batch-quota":
		s.checkBatchQuota()
	case "all":
		s.checkStaleLeads()
		s.expireLeads()
		s.expireInvoices()
		s.checkBatchQuota()
	default:
		return fmt.Errorf("unknown job %q (pilih: stale-leads|expire-leads|expire-invoices|batch-quota|all)", name)
	}
	return nil
}

// markNotified records that a notification of this kind went out for a
// reference, without there having been a WhatsApp send to record it.
//
// The dedup guards in this file all read wa_notifications, and that row is
// normally written as a side effect of sending. When there is nothing to send —
// a participant with no number — the marker has to be written anyway, or the job
// re-runs the rest of its work for that row every day forever.
func (s *Scheduler) markNotified(ctx context.Context, referenceID, messageType, recipientName string) {
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO wa_notifications
		(id,recipient_phone,recipient_name,message_type,message_content,reference_id,reference_type,status,created_at)
		VALUES (gen_random_uuid(),'-',$1,$2,$3,$4,'invoice','skipped',NOW())`,
		recipientName, messageType,
		"Tidak dikirim via WA: peserta tidak memiliki nomor WhatsApp", referenceID)
}

// getAdmins returns the users who receive admin notifications (§17.2.2).
func (s *Scheduler) getAdmins(ctx context.Context) []domainUser.User {
	return domainUser.ListAdmins(ctx, s.users)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// rupiah formats a float as a thousands-separated string (no symbol).
func rupiah(amount float64) string {
	s := fmt.Sprintf("%.0f", amount)
	n := len(s)
	out := make([]byte, 0, n+n/3)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, s[i])
	}
	return string(out)
}
