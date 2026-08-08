package httpdelivery

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/airport"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

type AirportHandler struct {
	repo         airport.Repository
	participants participant.Repository
	fonnte       *service.FonnteService
	pdf          *service.PDFService
}

func NewAirportHandler(
	repo airport.Repository,
	participants participant.Repository,
	fonnte *service.FonnteService,
	pdf *service.PDFService,
) *AirportHandler {
	return &AirportHandler{repo: repo, participants: participants, fonnte: fonnte, pdf: pdf}
}

// ListChecklist godoc
// @Summary      Daftar checklist airport per batch
// @Tags         airport
// @Security     BearerAuth
// @Param        batch_id query string true "Batch ID"
// @Param        status query string false "Filter status (pending/done)"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/airport/checklist [get]
func (h *AirportHandler) ListChecklist(c echo.Context) error {
	batchID := c.QueryParam("batch_id")
	if batchID == "" {
		return badRequest(c, "batch_id harus diisi")
	}
	f := airport.Filter{BatchID: batchID}
	if v := c.QueryParam("status"); v != "" {
		f.Status = &v
	}
	_ = h.repo.InitForBatch(c.Request().Context(), batchID)
	list, err := h.repo.ListByBatch(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	progress, _ := h.repo.GetBatchProgress(c.Request().Context(), batchID)
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"checklists": list,
		"progress":   progress,
	}))
}

// UpdateBaggage godoc
// @Summary      Tandai bagasi peserta sudah dicheck (FR-AIR-02)
// @Tags         airport
// @Security     BearerAuth
// @Param        participant_id path string true "Participant ID"
// @Param        batch_id query string true "Batch ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/airport/participants/{participant_id}/baggage [patch]
func (h *AirportHandler) UpdateBaggage(c echo.Context) error {
	batchID := c.QueryParam("batch_id")
	if batchID == "" {
		return badRequest(c, "batch_id harus diisi")
	}
	if err := h.repo.UpdateBaggage(c.Request().Context(), c.Param("participant_id"), batchID, claimUserID(c)); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(nil))
}

// UpdateTicket godoc
// @Summary      Tandai tiket peserta sudah didistribusikan (FR-AIR-02)
// @Tags         airport
// @Security     BearerAuth
// @Param        participant_id path string true "Participant ID"
// @Param        batch_id query string true "Batch ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/airport/participants/{participant_id}/ticket [patch]
func (h *AirportHandler) UpdateTicket(c echo.Context) error {
	batchID := c.QueryParam("batch_id")
	if batchID == "" {
		return badRequest(c, "batch_id harus diisi")
	}
	if err := h.repo.UpdateTicket(c.Request().Context(), c.Param("participant_id"), batchID, claimUserID(c)); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(nil))
}

// UpdatePassport godoc
// @Summary      Tandai paspor peserta sudah dikembalikan (FR-AIR-02/03)
// @Tags         airport
// @Security     BearerAuth
// @Param        participant_id path string true "Participant ID"
// @Param        batch_id query string true "Batch ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/airport/participants/{participant_id}/passport [patch]
func (h *AirportHandler) UpdatePassport(c echo.Context) error {
	batchID := c.QueryParam("batch_id")
	if batchID == "" {
		return badRequest(c, "batch_id harus diisi")
	}
	if err := h.repo.UpdatePassport(c.Request().Context(), c.Param("participant_id"), batchID, claimUserID(c)); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(nil))
}

// GetReport returns a JSON summary report for a batch (FR-AIR-06).
func (h *AirportHandler) GetReport(c echo.Context) error {
	batchID := c.QueryParam("batch_id")
	if batchID == "" {
		return badRequest(c, "batch_id harus diisi")
	}
	f := airport.Filter{BatchID: batchID}
	list, err := h.repo.ListByBatch(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	progress, _ := h.repo.GetBatchProgress(c.Request().Context(), batchID)
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"batch_id":     batchID,
		"generated_at": time.Now().Format(time.RFC3339),
		"progress":     progress,
		"checklists":   list,
	}))
}

// GetReportPDF godoc
// @Summary      Laporan post-handling sebagai PDF (FR-AIR-06)
// @Tags         airport
// @Security     BearerAuth
// @Param        batch_id query string true "Batch ID"
// @Produce      application/pdf
// @Success      200 {file} binary
// @Router       /admin/airport/report/pdf [get]
func (h *AirportHandler) GetReportPDF(c echo.Context) error {
	batchID := c.QueryParam("batch_id")
	if batchID == "" {
		return badRequest(c, "batch_id harus diisi")
	}
	f := airport.Filter{BatchID: batchID}
	list, err := h.repo.ListByBatch(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	progress, _ := h.repo.GetBatchProgress(c.Request().Context(), batchID)

	rows := make([]service.AirportRow, 0, len(list))
	tourLeaderName := "—"
	var startedAt, finishedAt *time.Time
	considerTime := func(t *time.Time) {
		if t == nil {
			return
		}
		if startedAt == nil || t.Before(*startedAt) {
			tt := *t
			startedAt = &tt
		}
		if finishedAt == nil || t.After(*finishedAt) {
			tt := *t
			finishedAt = &tt
		}
	}
	for _, cl := range list {
		fmtTime := func(t *time.Time) string {
			if t == nil {
				return "—"
			}
			return t.Format("15:04")
		}
		rows = append(rows, service.AirportRow{
			ParticipantName: cl.ParticipantName,
			BaggageAt:       fmtTime(cl.BaggageCheckedAt),
			TicketAt:        fmtTime(cl.TicketDistributedAt),
			PassportAt:      fmtTime(cl.PassportReturnedAt),
		})
		if cl.HandledByName != nil && *cl.HandledByName != "" {
			tourLeaderName = *cl.HandledByName
		}
		considerTime(cl.BaggageCheckedAt)
		considerTime(cl.TicketDistributedAt)
		considerTime(cl.PassportReturnedAt)
	}

	batchName := batchID[:8]
	departureDate := ""
	if len(list) > 0 {
		batchName = "Batch " + batchName
	}

	data := service.AirportReportData{
		BatchName:      batchName,
		DepartureDate:  departureDate,
		TourLeaderName: tourLeaderName,
		TotalPax:       progress.TotalPax,
		DoneCount:      progress.DoneCount,
		PendingCount:   progress.PendingCount,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		GeneratedAt:    time.Now(),
		Rows:           rows,
	}

	pdfBytes, err := h.pdf.GenerateAirportReport(data)
	if err != nil {
		return serverErr(c, err)
	}
	c.Response().Header().Set("Content-Disposition", "attachment; filename=airport-report.pdf")
	return c.Blob(http.StatusOK, "application/pdf", pdfBytes)
}

// ConfirmDeparture godoc
// @Summary      Kirim WA konfirmasi keberangkatan ke semua peserta batch (FR-AUTO-08)
// @Tags         airport
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /admin/airport/confirm-departure [post]
func (h *AirportHandler) ConfirmDeparture(c echo.Context) error {
	var body struct {
		BatchID      string `json:"batch_id"`
		GatherPoint  string `json:"gather_point"`
		GatherTime   string `json:"gather_time"`
		Gate         string `json:"gate"`
		CheckinTime  string `json:"checkin_time"`
	}
	if err := bindJSON(c, &body); err != nil || body.BatchID == "" {
		return badRequest(c, "batch_id harus diisi")
	}

	// Get all participants in this batch
	pts, err := h.participants.ListByBatch(c.Request().Context(), body.BatchID)
	if err != nil {
		return serverErr(c, err)
	}

	progress, _ := h.repo.GetBatchProgress(c.Request().Context(), body.BatchID)
	if progress != nil && progress.PendingCount > 0 {
		return c.JSON(http.StatusUnprocessableEntity, errResponse("NOT_COMPLETE",
			fmt.Sprintf("Masih ada %d peserta yang belum selesai diproses", progress.PendingCount)))
	}

	// Send WA blast in background
	safe.Go("blast WA konfirmasi keberangkatan", func() {
		bgCtx := context.Background()
		for _, p := range pts {
			msg := fmt.Sprintf(
				"✈️ *KONFIRMASI KEBERANGKATAN*\n\n"+
					"Halo *%s*!\n\n"+
					"Airport handling untuk batch Anda telah selesai. Berikut informasi keberangkatan:\n\n"+
					"📍 Titik Kumpul: *%s*\n"+
					"🕐 Waktu Kumpul: *%s*\n"+
					"🚪 Gate: *%s*\n"+
					"⏰ Check-in: *%s*\n\n"+
					"Selamat menikmati perjalanan! 🌟\n\n"+
					"_Tim Pintour Travel_",
				p.Name,
				ifEmpty(body.GatherPoint, "Sesuai briefing"),
				ifEmpty(body.GatherTime, "Sesuai jadwal"),
				ifEmpty(body.Gate, "Lihat papan informasi"),
				ifEmpty(body.CheckinTime, "Sesuai tiket"),
			)
			refType := "batch"
			_ = h.fonnte.Send(bgCtx, p.Phone, p.Name,
				notification.TypeDepartureConfirm, msg, &body.BatchID, &refType)
			time.Sleep(time.Second)
		}
	})

	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"message":      fmt.Sprintf("WA konfirmasi dikirim ke %d peserta", len(pts)),
		"participants": len(pts),
	}))
}

func ifEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
