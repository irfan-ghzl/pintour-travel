package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2" // PRD §18: gocron untuk cron in-process

	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// Scheduler runs automated WA notification jobs (§17) + retention cleanup (§25.4)
// + the automation jobs (prompt §2) via gocron (PRD §18 — go-co-op/gocron).
type Scheduler struct {
	participants participant.Repository
	users        domainUser.Repository
	fonnte       *service.FonnteService
	email        *service.EmailService
	db           *sql.DB
	cron         gocron.Scheduler
}

func New(participants participant.Repository, users domainUser.Repository, fonnte *service.FonnteService, email *service.EmailService, db *sql.DB) (*Scheduler, error) {
	cron, err := gocron.NewScheduler(
		gocron.WithLocation(time.Local),
	)
	if err != nil {
		return nil, fmt.Errorf("init gocron: %w", err)
	}
	return &Scheduler{participants: participants, users: users, fonnte: fonnte, email: email, db: db, cron: cron}, nil
}

// Start launches all scheduled jobs via gocron.
func (s *Scheduler) Start() {
	jobs := []struct {
		atTime string
		fn     func()
		label  string
	}{
		{"09:00", s.sendPaymentReminders, "payment-reminders H+1/H+3/H+6"},
		{"08:00", s.sendDepartureReminders, "departure-reminders H-30/H-14/H-7/H-1"},
		{"00:01", s.activateBriefing, "briefing-activation H-14"},
		{"03:00", s.sendAirportInfo, "airport-info hari-H"},
		{"02:00", s.retentionCleanup, "retention-cleanup §25.4"},
		// Automation jobs (prompt §2)
		{"00:01", s.expireLeads, "expire-leads §2.2"},
		{"00:05", s.expireInvoices, "expire-invoices §2.3"},
		{"08:00", s.checkBatchQuota, "batch-quota-warning §2.4"},
	}

	for _, j := range jobs {
		t, _ := time.Parse("15:04", j.atTime)
		_, err := s.cron.NewJob(
			gocron.DailyJob(1, gocron.NewAtTimes(
				gocron.NewAtTime(uint(t.Hour()), uint(t.Minute()), 0),
			)),
			// gocron recovers a panicking job but keeps only the value; wrapping
			// it means the stack that caused it reaches the log as well, in the
			// same shape as every other background job.
			gocron.NewTask(safe.Recovered(j.label, j.fn)),
			gocron.WithName(j.label),
		)
		if err != nil {
			log.Printf("scheduler: failed to schedule %s: %v", j.label, err)
		}
	}

	// checkStaleLeads runs hourly (prompt §2.1).
	if _, err := s.cron.NewJob(
		gocron.DurationJob(time.Hour),
		gocron.NewTask(safe.Recovered("stale-leads §2.1", s.checkStaleLeads)),
		gocron.WithName("stale-leads §2.1"),
	); err != nil {
		log.Printf("scheduler: failed to schedule stale-leads: %v", err)
	}

	s.cron.Start()
	log.Println("Scheduler started (gocron): WA jobs + retention cleanup + automation §2")
}

// Stop shuts down the scheduler gracefully.
func (s *Scheduler) Stop() {
	if s.cron != nil {
		_ = s.cron.Shutdown()
	}
}

// sendPaymentReminders sends WA to participants with unpaid invoices (H+1, H+3, H+6).
func (s *Scheduler) sendPaymentReminders() {
	ctx := context.Background()
	reminders := []struct {
		days     int
		dayLabel string
	}{
		{1, "1 hari"},
		{3, "3 hari"},
		{6, "6 hari"},
	}
	for _, r := range reminders {
		// Read the invoice, not just the participant. The old call passed an empty
		// invoice number — so the message named no invoice, leaving a participant
		// with two unpaid ones no way to tell which was meant — and passed the
		// PARTICIPANT id as the reference while labelling it "invoice", which
		// broke the trail from an invoice to the messages sent about it.
		for _, due := range s.unpaidInvoiceReminders(ctx, r.days) {
			if due.phone == "" {
				continue
			}
			_ = s.fonnte.SendPaymentReminder(ctx, due.phone, due.name,
				due.invoiceNumber, r.dayLabel, due.invoiceID)
			time.Sleep(time.Second) // §17.2 rate-limit Fonnte
		}
	}
}

// unpaidInvoiceReminder is one participant to nudge about one invoice.
type unpaidInvoiceReminder struct {
	invoiceID     string
	invoiceNumber string
	name          string
	phone         string
}

// unpaidInvoiceReminders lists the unpaid invoices issued `days` days ago along
// with who to tell, deduped against a reminder of the same kind already sent for
// that invoice today — the same marker pattern the overdue and quota jobs use.
func (s *Scheduler) unpaidInvoiceReminders(ctx context.Context, days int) []unpaidInvoiceReminder {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT i.id, i.invoice_number, p.name, COALESCE(p.phone,'')
		FROM invoices i
		JOIN participants p ON p.id = i.participant_id
		WHERE i.deleted_at IS NULL
		  AND i.status IN ('diterbitkan','menunggu_bayar')
		  AND i.created_at::date = (NOW() - ($1 || ' days')::interval)::date
		ORDER BY i.created_at`, days)
	if err != nil {
		log.Printf("sendPaymentReminders query (H+%d): %v", days, err)
		return nil
	}
	defer rows.Close()

	var out []unpaidInvoiceReminder
	for rows.Next() {
		var r unpaidInvoiceReminder
		if err := rows.Scan(&r.invoiceID, &r.invoiceNumber, &r.name, &r.phone); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

// sendDepartureReminders sends WA blasts at H-30/H-14/H-7/H-1.
func (s *Scheduler) sendDepartureReminders() {
	ctx := context.Background()
	type reminder struct {
		days    int
		label   string
		msgType string
	}
	reminders := []reminder{
		{30, "H-30", notification.TypeReminderH30},
		{14, "H-14", notification.TypeReminderH14},
		{7, "H-7", notification.TypeReminderH7},
		{1, "H-1", notification.TypeReminderH1},
	}
	for _, r := range reminders {
		pts, err := s.participants.ListByDepartureDaysAhead(ctx, r.days)
		if err != nil {
			continue
		}
		for _, p := range pts {
			depDate := ""
			if p.BatchDepartureDate != nil {
				depDate = p.BatchDepartureDate.Format("02 Jan 2006")
			}
			_ = s.fonnte.SendDepartureReminder(ctx, p.Phone, p.Name,
				p.PackageName, depDate, r.label, r.msgType, p.ID)
			// §3.2 H-14 reminder email (checklist persiapan).
			if r.days == 14 && s.email != nil && p.Email != "" {
				_ = s.email.SendEmailReminderH14(ctx, p.Email, p.Name, p.PackageName, depDate,
					envOr("PORTAL_BASE_URL", "http://localhost:3000")+"/portal")
			}
			time.Sleep(time.Second)
		}
	}
}

// activateBriefing notifies participants that briefing is active (H-14).
func (s *Scheduler) activateBriefing() {
	ctx := context.Background()
	pts, err := s.participants.ListByDepartureDaysAhead(ctx, 14)
	if err != nil {
		return
	}
	for _, p := range pts {
		depDate := ""
		if p.BatchDepartureDate != nil {
			depDate = p.BatchDepartureDate.Format("02 Jan 2006")
		}
		msg := fmt.Sprintf(
			"📖 *Briefing Digital Sudah Aktif!*\n\n"+
				"Halo *%s*! Perjalanan *%s* pada *%s* akan segera tiba.\n\n"+
				"Fitur Briefing Digital sudah aktif di portal peserta Anda. "+
				"Silakan login dan baca materi briefing untuk persiapan keberangkatan.\n\n"+
				"🔗 Portal: /portal/briefing\n\n"+
				"_Tim Pintour Travel_",
			p.Name, p.PackageName, depDate,
		)
		refType := "participant"
		_ = s.fonnte.Send(ctx, p.Phone, p.Name,
			notification.TypeReminderH14, msg, &p.ID, &refType)
		// §3.2 briefing-activated email.
		if s.email != nil && p.Email != "" {
			_ = s.email.SendEmailBriefingActivated(ctx, p.Email, p.Name, p.PackageName, "",
				depDate, envOr("PORTAL_BASE_URL", "http://localhost:3000")+"/portal/briefing")
		}
		time.Sleep(time.Second)
	}
}

// sendAirportInfo sends full departure day info to participants at 03:00.
func (s *Scheduler) sendAirportInfo() {
	ctx := context.Background()
	pts, err := s.participants.ListByDepartureDaysAhead(ctx, 0)
	if err != nil {
		return
	}
	for _, p := range pts {
		depDate := "hari ini"
		if p.BatchDepartureDate != nil {
			depDate = p.BatchDepartureDate.Format("02 Jan 2006")
		}
		msg := fmt.Sprintf(
			"✈️ *HARI KEBERANGKATAN — %s*\n\n"+
				"Halo *%s*! Selamat, hari ini Anda berangkat untuk perjalanan *%s*!\n\n"+
				"📋 *Checklist terakhir:*\n"+
				"✓ Paspor & dokumen perjalanan\n"+
				"✓ Tiket pesawat (cek email/portal)\n"+
				"✓ Bagasi sudah siap\n"+
				"✓ Baterai HP penuh\n\n"+
				"🕐 Harap tiba di titik kumpul 3 jam sebelum jadwal penerbangan.\n\n"+
				"Tour leader Anda akan segera menghubungi grup WhatsApp.\n"+
				"Kontak darurat: hubungi tim Pintour jika ada kendala.\n\n"+
				"Selamat berlibur! 🌟",
			depDate, p.Name, p.PackageName,
		)
		refType := "participant"
		_ = s.fonnte.Send(ctx, p.Phone, p.Name,
			notification.TypeDepartureConfirm, msg, &p.ID, &refType)
		time.Sleep(time.Second)
	}
}

// retentionCleanup enforces PRD §25.4 data retention policy.
func (s *Scheduler) retentionCleanup() {
	if s.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	jobs := []struct {
		name string
		sql  string
	}{
		{
			name: "soft-delete leads tidak_deal > 1 tahun",
			sql: `UPDATE leads SET deleted_at=NOW()
				WHERE status='tidak_deal' AND created_at < NOW() - INTERVAL '1 year'
				AND deleted_at IS NULL`,
		},
		{
			name: "soft-delete dokumen sensitif > 1 tahun setelah keberangkatan",
			sql: `UPDATE documents d SET deleted_at=NOW()
				FROM participants p JOIN package_batches pb ON pb.id=p.batch_id
				WHERE d.participant_id=p.id
				AND d.document_type IN ('passport','ktp','rekening_koran')
				AND pb.return_date < NOW() - INTERVAL '1 year'
				AND d.deleted_at IS NULL`,
		},
		{
			name: "hard-delete log WA > 2 tahun",
			sql:  `DELETE FROM wa_notifications WHERE created_at < NOW() - INTERVAL '2 years'`,
		},
	}

	for _, j := range jobs {
		res, err := s.db.ExecContext(ctx, j.sql)
		if err != nil {
			log.Printf("retention[%s] error: %v", j.name, err)
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			log.Printf("retention[%s]: %d rows", j.name, n)
		}
	}
}
