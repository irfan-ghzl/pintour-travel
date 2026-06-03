-- ============================================================
-- Migration 005: Tambah facilities ke packages (FR-CAT-03)
-- ============================================================

-- FR-CAT-03: "Halaman detail menampilkan: itinerary, fasilitas yang disertakan,
-- syarat & ketentuan, informasi visa, harga per kategori"
ALTER TABLE packages ADD COLUMN IF NOT EXISTS facilities JSONB NOT NULL DEFAULT '[]';
ALTER TABLE packages ADD COLUMN IF NOT EXISTS terms_conditions TEXT;
ALTER TABLE packages ADD COLUMN IF NOT EXISTS visa_info TEXT;

-- Index untuk filter durasi (FR-CAT-02)
CREATE INDEX IF NOT EXISTS idx_packages_duration ON packages(duration_days) WHERE deleted_at IS NULL;

-- Index untuk filter tanggal masuk leads (FR-CRM-05)
CREATE INDEX IF NOT EXISTS idx_leads_created_at ON leads(created_at) WHERE deleted_at IS NULL;
