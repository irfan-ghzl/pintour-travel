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
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// Scheduler runs automated WA notification jobs (§17) + retention cleanup (§25.4)
// via gocron (PRD §18 — go-co-op/gocron).
type Scheduler struct {
	participants participant.Repository
	fonnte       *service.FonnteService
	db           *sql.DB
	cron         gocron.Scheduler
}

func New(participants participant.Repository, fonnte *service.FonnteService, db *sql.DB) (*Scheduler, error) {
	cron, err := gocron.NewScheduler(
		gocron.WithLocation(time.Local),
	)
	if err != nil {
		return nil, fmt.Errorf("init gocron: %w", err)
	}
	return &Scheduler{participants: participants, fonnte: fonnte, db: db, cron: cron}, nil
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
	}

	for _, j := range jobs {
		t, _ := time.Parse("15:04", j.atTime)
		_, err := s.cron.NewJob(
			gocron.DailyJob(1, gocron.NewAtTimes(
				gocron.NewAtTime(uint(t.Hour()), uint(t.Minute()), 0),
			)),
			gocron.NewTask(j.fn),
			gocron.WithName(j.label),
		)
		if err != nil {
			log.Printf("scheduler: failed to schedule %s: %v", j.label, err)
		}
	}

	s.cron.Start()
	log.Println("Scheduler started (gocron): WA jobs + retention cleanup")
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
		pts, err := s.participants.ListWithUnpaidInvoiceDaysOld(ctx, r.days)
		if err != nil {
			continue
		}
		for _, p := range pts {
			_ = s.fonnte.SendPaymentReminder(ctx, p.Phone, p.Name, "", r.dayLabel, p.ID)
			time.Sleep(time.Second) // §17.2 rate-limit Fonnte
		}
	}
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
