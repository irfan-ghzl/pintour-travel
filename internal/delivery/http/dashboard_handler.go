package httpdelivery

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// DashboardHandler handles admin dashboard requests.
type DashboardHandler struct{ db *sql.DB }

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// GetStats godoc
//
//	@Summary     Dashboard statistics (admin)
//	@Tags        dashboard
//	@Produce     json
//	@Security    BearerAuth
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/dashboard/stats [get]
func (h *DashboardHandler) GetStats(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"generated_at": time.Now().Format(time.RFC3339),
		"note":         "Gunakan /admin/dashboard/analytics untuk metrik lengkap",
	})
}

// GetAnalytics godoc
//
//	@Summary     Real-time dashboard analytics (prompt §4.3)
//	@Tags        dashboard
//	@Produce     json
//	@Security    BearerAuth
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/dashboard/analytics [get]
func (h *DashboardHandler) GetAnalytics(c echo.Context) error {
	if h.db == nil {
		return serverErr(c, sql.ErrConnDone)
	}
	ctx := c.Request().Context()

	// ── Leads summary ─────────────────────────────────────────────────────────
	var total, baru, proses, deal, expired int
	_ = h.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		  COUNT(*) FILTER (WHERE status='baru'),
		  COUNT(*) FILTER (WHERE status IN ('dihubungi','konsultasi')),
		  COUNT(*) FILTER (WHERE status IN ('deal','peserta')),
		  COUNT(*) FILTER (WHERE status='tidak_deal')
		FROM leads WHERE deleted_at IS NULL`).Scan(&total, &baru, &proses, &deal, &expired)
	conversion := 0.0
	if total > 0 {
		conversion = float64(deal) / float64(total) * 100
	}

	// ── Revenue summary ───────────────────────────────────────────────────────
	var invoiced, paid, pending, overdue float64
	_ = h.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount),0),
		  COALESCE(SUM(amount) FILTER (WHERE status='lunas'),0),
		  COALESCE(SUM(amount) FILTER (WHERE status IN ('diterbitkan','menunggu_bayar','dibayar')),0),
		  COALESCE(SUM(amount) FILTER (WHERE status IN ('diterbitkan','menunggu_bayar') AND due_date < NOW()),0)
		FROM invoices WHERE deleted_at IS NULL`).Scan(&invoiced, &paid, &pending, &overdue)

	// ── Batch summary ─────────────────────────────────────────────────────────
	var activeBatches, totalParticipants int
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM package_batches WHERE status='tersedia'`).Scan(&activeBatches)
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM participants WHERE is_active=true`).Scan(&totalParticipants)

	var nearest interface{}
	{
		var batchID, pkgName string
		var depDate time.Time
		var daysRemaining, paxCount int
		err := h.db.QueryRowContext(ctx, `
			SELECT pb.id, pkg.name, pb.departure_date,
			  (pb.departure_date - CURRENT_DATE),
			  COALESCE((SELECT COUNT(*) FROM participants p WHERE p.batch_id=pb.id),0)
			FROM package_batches pb
			JOIN packages pkg ON pkg.id = pb.package_id
			WHERE pb.departure_date >= CURRENT_DATE
			ORDER BY pb.departure_date ASC
			LIMIT 1`).Scan(&batchID, &pkgName, &depDate, &daysRemaining, &paxCount)
		if err == nil {
			nearest = map[string]interface{}{
				"batch_id":          batchID,
				"package_name":      pkgName,
				"departure_date":    depDate.Format("2006-01-02"),
				"days_remaining":    daysRemaining,
				"participant_count": paxCount,
			}
		}
	}

	// ── Monthly leads (last 6 months) ─────────────────────────────────────────
	monthly := []map[string]interface{}{}
	rows, err := h.db.QueryContext(ctx, `
		SELECT to_char(date_trunc('month', created_at), 'Mon YYYY') AS month, COUNT(*)
		FROM leads
		WHERE deleted_at IS NULL
		  AND created_at >= date_trunc('month', NOW()) - INTERVAL '5 months'
		GROUP BY date_trunc('month', created_at)
		ORDER BY date_trunc('month', created_at)`)
	if err == nil {
		for rows.Next() {
			var month string
			var count int
			if err := rows.Scan(&month, &count); err == nil {
				monthly = append(monthly, map[string]interface{}{"month": month, "count": count})
			}
		}
		rows.Close()
	}

	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"leads_summary": map[string]interface{}{
			"total": total, "baru": baru, "proses": proses, "deal": deal,
			"expired": expired, "conversion_rate": conversion,
		},
		"revenue_summary": map[string]interface{}{
			"total_invoiced": invoiced, "total_paid": paid,
			"total_pending": pending, "total_overdue": overdue,
		},
		"batch_summary": map[string]interface{}{
			"total_active_batches": activeBatches,
			"total_participants":   totalParticipants,
			"nearest_departure":    nearest,
		},
		"monthly_leads": monthly,
	}))
}
