-- ============================================================
-- Migration 008: Lead status audit trail + soft delete completion
--   FR-CRM-02: "Setiap perubahan status harus tercatat dengan timestamp
--               dan pengguna yang melakukan perubahan"
--   §13.1 / ERD §14.1: hapus lunak berlaku pada seluruh tabel transaksional
-- ============================================================

-- ── Riwayat status lead (FR-CRM-02) ─────────────────────────────────────────
-- Sebelumnya tidak ada tabel ini sama sekali, dan parameter pelaku pada
-- repository dibuang begitu saja — sehingga "siapa memindahkan lead ini ke
-- deal?" tidak terjawab di manapun.
--
-- changed_by NULL berarti perubahan dilakukan sistem (penjadwal), bukan
-- pengguna yang datanya hilang: pengguna dihapus lunak, tidak pernah dihapus
-- fisik, jadi ON DELETE SET NULL di bawah tidak pernah benar-benar terpicu.
CREATE TABLE IF NOT EXISTS lead_status_history (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id     UUID        NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_status VARCHAR(20),
    to_status   VARCHAR(20) NOT NULL,
    changed_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Dibaca selalu per lead dan berurut waktu (panel log aktivitas admin).
CREATE INDEX IF NOT EXISTS idx_lead_status_history_lead
    ON lead_status_history(lead_id, changed_at);

-- ── Hapus lunak pada sisa tabel transaksional (§13.1, ERD §14.1) ────────────
-- Migrasi 004 sudah menangani packages/leads/participants/invoices/documents.
-- Dua ini tertinggal, dan keduanya transaksional: bukti bayar adalah catatan
-- keuangan, checklist bandara adalah catatan operasional keberangkatan.
ALTER TABLE payment_proofs     ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE airport_checklists ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Pembacaan keduanya selalu menyaring baris terhapus, jadi indeksnya parsial.
CREATE INDEX IF NOT EXISTS idx_payment_proofs_invoice_active
    ON payment_proofs(invoice_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_airport_checklists_batch_active
    ON airport_checklists(batch_id) WHERE deleted_at IS NULL;

-- Catatan penomoran invoice (§13.7): nomor milik baris terhapus lunak TIDAK
-- boleh dipakai ulang, jadi kueri NextSequence sengaja tidak menyaring
-- deleted_at — satu-satunya pembacaan invoices yang memang begitu.
