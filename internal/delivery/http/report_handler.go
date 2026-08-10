package httpdelivery

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"

	"github.com/irfan-ghzl/pintour-travel/internal/format"
)

// ReportHandler exports admin reports as Excel or PDF (prompt §4.1 / §4.2).
type ReportHandler struct{ db *sql.DB }

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(db *sql.DB) *ReportHandler { return &ReportHandler{db: db} }

// reportData is the generic shape every report type resolves to.
type reportData struct {
	sheet   string
	title   string
	headers []string
	rows    [][]string
}

// Export godoc
//
//	@Summary     Export laporan (leads/participants/invoices/batches) ke Excel/PDF
//	@Tags        reports
//	@Produce     application/octet-stream
//	@Security    BearerAuth
//	@Param       type   query string true "leads|participants|invoices|batches"
//	@Param       format query string true "excel|pdf"
//	@Success     200 {file} binary
//	@Router      /admin/reports/export [get]
func (h *ReportHandler) Export(c echo.Context) error {
	if h.db == nil {
		return serverErr(c, sql.ErrConnDone)
	}
	typ := c.QueryParam("type")
	fileFormat := c.QueryParam("format")
	if fileFormat == "" {
		fileFormat = "excel"
	}

	data, err := h.resolve(c.Request().Context(), typ)
	if err != nil {
		if err == errUnknownReport {
			return badRequest(c, "type harus salah satu dari: leads, participants, invoices, batches")
		}
		return serverErr(c, err)
	}

	filename := fmt.Sprintf("pintour-%s-%s", typ, time.Now().Format("20060102"))
	switch fileFormat {
	case "pdf":
		return h.writePDF(c, data, filename+".pdf")
	default:
		return h.writeExcel(c, data, filename+".xlsx")
	}
}

var errUnknownReport = fmt.Errorf("unknown report type")

func (h *ReportHandler) resolve(ctx context.Context, typ string) (*reportData, error) {
	switch typ {
	case "leads":
		return h.queryRows(ctx, &reportData{
			sheet: "Leads", title: "Laporan Leads",
			headers: []string{"No", "Nama", "Nomor HP", "Email", "Paket", "Batch", "Status", "Tanggal Masuk", "Konsultan"},
		}, `
			SELECT l.name, l.phone, COALESCE(l.email,''), COALESCE(pkg.name,''),
			  COALESCE(to_char(pb.departure_date,'YYYY-MM-DD'),'-'), l.status,
			  to_char(l.created_at,'YYYY-MM-DD'), COALESCE(u.name,'-')
			FROM leads l
			LEFT JOIN packages pkg ON pkg.id=l.package_id
			LEFT JOIN package_batches pb ON pb.id=l.batch_id
			LEFT JOIN users u ON u.id=l.assigned_to
			WHERE l.deleted_at IS NULL
			ORDER BY l.created_at DESC`)
	case "participants":
		return h.queryRows(ctx, &reportData{
			sheet: "Peserta", title: "Laporan Peserta",
			headers: []string{"No", "Nama", "Paket", "Tanggal Berangkat", "Status Bayar", "Status Dokumen", "Status"},
		}, `
			SELECT p.name, COALESCE(pkg.name,''),
			  COALESCE(to_char(pb.departure_date,'YYYY-MM-DD'),'-'),
			  COALESCE((SELECT i.status FROM invoices i WHERE i.participant_id=p.id AND i.deleted_at IS NULL ORDER BY i.created_at DESC LIMIT 1),'-'),
			  COALESCE((SELECT CASE WHEN COUNT(*)=0 THEN 'belum ada'
			    WHEN COUNT(*) FILTER (WHERE status='disetujui')=COUNT(*) THEN 'lengkap'
			    ELSE 'proses' END FROM documents d WHERE d.participant_id=p.id),'belum ada'),
			  CASE WHEN p.is_active THEN 'aktif' ELSE 'nonaktif' END
			FROM participants p
			LEFT JOIN package_batches pb ON pb.id=p.batch_id
			LEFT JOIN packages pkg ON pkg.id=pb.package_id
			ORDER BY p.created_at DESC`)
	case "invoices":
		return h.queryRows(ctx, &reportData{
			sheet: "Invoice", title: "Laporan Invoice",
			headers: []string{"No Invoice", "Nama Peserta", "Paket", "Nominal", "Terbayar", "Sisa", "Status", "Due Date"},
		}, `
			SELECT i.invoice_number, p.name, COALESCE(pkg.name,''),
			  i.amount,
			  COALESCE((SELECT SUM(pp.amount_claimed) FROM payment_proofs pp WHERE pp.invoice_id=i.id AND pp.status='disetujui' AND pp.deleted_at IS NULL),0),
			  i.status, to_char(i.due_date,'YYYY-MM-DD')
			FROM invoices i
			JOIN participants p ON p.id=i.participant_id
			JOIN package_batches pb ON pb.id=i.batch_id
			JOIN packages pkg ON pkg.id=pb.package_id
			WHERE i.deleted_at IS NULL
			ORDER BY i.created_at DESC`)
	case "batches":
		return h.queryRows(ctx, &reportData{
			sheet: "Batch Keberangkatan", title: "Laporan Batch Keberangkatan",
			headers: []string{"Paket", "Tour Leader", "Tgl Berangkat", "Total Kuota", "Terjual", "Sisa", "Status"},
		}, `
			SELECT pkg.name, COALESCE(u.name,'-'), to_char(pb.departure_date,'YYYY-MM-DD'),
			  pb.quota,
			  COALESCE((SELECT COUNT(*) FROM leads l WHERE l.batch_id=pb.id AND l.status='deal'),0),
			  pb.status
			FROM package_batches pb
			JOIN packages pkg ON pkg.id=pb.package_id
			LEFT JOIN users u ON u.id=pb.tour_leader_id
			ORDER BY pb.departure_date`)
	}
	return nil, errUnknownReport
}

// queryRows runs the report query and post-processes the rows into display
// strings. The column count expected per row is len(headers) minus computed
// columns; each report below documents its derived columns.
func (h *ReportHandler) queryRows(ctx context.Context, rd *reportData, query string) (*reportData, error) {
	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idx := 1
	for rows.Next() {
		switch rd.sheet {
		case "Leads":
			var name, phone, email, pkg, batch, status, created, consultant string
			if err := rows.Scan(&name, &phone, &email, &pkg, &batch, &status, &created, &consultant); err != nil {
				return nil, err
			}
			rd.rows = append(rd.rows, []string{fmt.Sprint(idx), name, phone, email, pkg, batch, status, created, consultant})
		case "Peserta":
			var name, pkg, dep, bayar, dok, status string
			if err := rows.Scan(&name, &pkg, &dep, &bayar, &dok, &status); err != nil {
				return nil, err
			}
			rd.rows = append(rd.rows, []string{fmt.Sprint(idx), name, pkg, dep, bayar, dok, status})
		case "Invoice":
			var num, name, pkg, status, due string
			var amount, paid float64
			if err := rows.Scan(&num, &name, &pkg, &amount, &paid, &status, &due); err != nil {
				return nil, err
			}
			rd.rows = append(rd.rows, []string{num, name, pkg, "Rp " + format.Rupiah(amount),
				"Rp " + format.Rupiah(paid), "Rp " + format.Rupiah(amount-paid), status, due})
		case "Batch Keberangkatan":
			var pkg, tl, dep, status string
			var quota, sold int
			if err := rows.Scan(&pkg, &tl, &dep, &quota, &sold, &status); err != nil {
				return nil, err
			}
			rd.rows = append(rd.rows, []string{pkg, tl, dep, fmt.Sprint(quota), fmt.Sprint(sold), fmt.Sprint(quota - sold), status})
		}
		idx++
	}
	return rd, rows.Err()
}

func (h *ReportHandler) writeExcel(c echo.Context, d *reportData, filename string) error {
	f := excelize.NewFile()
	defer f.Close()
	sheet := d.sheet
	idx, _ := f.NewSheet(sheet)
	f.SetActiveSheet(idx)
	f.DeleteSheet("Sheet1")

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4A7BA0"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Header row + column width tracking for auto-fit.
	widths := make([]int, len(d.headers))
	for i, head := range d.headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, head)
		widths[i] = len(head)
	}
	hdrCell, _ := excelize.CoordinatesToCellName(len(d.headers), 1)
	f.SetCellStyle(sheet, "A1", hdrCell, headerStyle)

	for r, row := range d.rows {
		for ci, val := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, r+2)
			f.SetCellValue(sheet, cell, val)
			if len(val) > widths[ci] {
				widths[ci] = len(val)
			}
		}
	}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		width := float64(w) + 2
		if width > 50 {
			width = 50
		}
		f.SetColWidth(sheet, col, col, width)
	}
	// Freeze the header row.
	f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", "attachment; filename="+filename)
	c.Response().WriteHeader(http.StatusOK)
	return f.Write(c.Response().Writer)
}

func (h *ReportHandler) writePDF(c echo.Context, d *reportData, filename string) error {
	pdf := gofpdf.New("L", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(0, 8, tr(fmt.Sprintf("Halaman %d — Digenerate oleh sistem Pintour", pdf.PageNo())),
			"", 0, "C", false, 0, "")
	})
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(16, 100, 59)
	pdf.CellFormat(0, 8, tr(d.title), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(0, 6, tr("Tanggal generate: "+time.Now().Format("02 Jan 2006 15:04")), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	usable := 277.0
	colW := usable / float64(len(d.headers))

	// Header
	pdf.SetFillColor(74, 123, 160)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	for _, head := range d.headers {
		pdf.CellFormat(colW, 7, tr(head), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Rows
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 7)
	fill := false
	for _, row := range d.rows {
		if fill {
			pdf.SetFillColor(244, 246, 248)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		for _, val := range row {
			pdf.CellFormat(colW, 6, tr(format.Ellipsis(val, 40)), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
		fill = !fill
	}

	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", "attachment; filename="+filename)
	c.Response().WriteHeader(http.StatusOK)
	return pdf.Output(c.Response().Writer)
}
