-- ============================================================
-- Migration 006: v2.0 features
--   F1 Payment Gateway (Midtrans) — invoices.snap_token/midtrans_order_id + status
--   F2 Chatbot AI                  — chatbot_logs
--   F3 Auto-OCR                    — document_ocr_results + participants.nik
-- ============================================================

-- ── F1: Midtrans Snap columns on invoices ────────────────────────────────────
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS snap_token         VARCHAR(255);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS midtrans_order_id  VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_invoices_midtrans_order ON invoices(midtrans_order_id)
  WHERE midtrans_order_id IS NOT NULL;

-- Allow the extra gateway-pending status used by the webhook.
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_status_check
  CHECK (status IN ('diterbitkan', 'menunggu_bayar', 'dibayar', 'lunas', 'menunggu_konfirmasi_gateway'));

-- ── F2: Chatbot conversation log ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS chatbot_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       VARCHAR(20)     NOT NULL,
    role        VARCHAR(10)     NOT NULL CHECK (role IN ('user', 'assistant')),
    message     TEXT            NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chatbot_logs_phone   ON chatbot_logs(phone);
CREATE INDEX IF NOT EXISTS idx_chatbot_logs_created ON chatbot_logs(created_at);

-- ── F3: OCR results + participant NIK ────────────────────────────────────────
ALTER TABLE participants ADD COLUMN IF NOT EXISTS nik VARCHAR(20);

CREATE TABLE IF NOT EXISTS document_ocr_results (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id        UUID            NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    extracted_data     JSONB           NOT NULL DEFAULT '{}',
    confidence         FLOAT           NOT NULL DEFAULT 0,
    validation_passed  BOOLEAN         NOT NULL DEFAULT FALSE,
    validation_notes   TEXT,
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ocr_results_document ON document_ocr_results(document_id);
