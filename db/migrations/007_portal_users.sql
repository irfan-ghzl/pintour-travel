-- ============================================================
-- Migration 007: Portal Users & Returning Customer (v2.0)
--   F1 Portal Users      — satu akun portal untuk banyak tour
--   F2 Riwayat Perjalanan — query semua participants per portal_user
--   F4 Returning Customer — leads.is_returning + portal_user_id
-- ============================================================

-- ── F1: portal_users (identitas portal terpusat) ─────────────────────────────
CREATE TABLE IF NOT EXISTS portal_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone         VARCHAR(20)  NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(100) NOT NULL,
    email         VARCHAR(100),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_portal_users_phone ON portal_users(phone);

-- ── F1: link participants → portal_users (nullable untuk backward compat) ─────
ALTER TABLE participants ADD COLUMN IF NOT EXISTS portal_user_id UUID
    REFERENCES portal_users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_participants_portal_user ON participants(portal_user_id);

-- ── F4: leads returning-customer metadata ────────────────────────────────────
ALTER TABLE leads ADD COLUMN IF NOT EXISTS portal_user_id UUID
    REFERENCES portal_users(id) ON DELETE SET NULL;
ALTER TABLE leads ADD COLUMN IF NOT EXISTS is_returning BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_leads_returning_assigned ON leads(is_returning, assigned_to)
    WHERE deleted_at IS NULL;

-- ── Backfill: buat portal_user dari participant terbaru per nomor telepon ─────
-- DISTINCT ON (phone) + ORDER BY created_at DESC mengambil password & nama paling
-- baru sebagai kanonik. portal_password lama dipindah jadi password_hash.
INSERT INTO portal_users (phone, password_hash, name, email, created_at)
SELECT DISTINCT ON (phone)
    phone, portal_password, name, email, created_at
FROM participants
WHERE deleted_at IS NULL
ORDER BY phone, created_at DESC
ON CONFLICT (phone) DO NOTHING;

-- Link semua participant lama ke portal_user yang baru dibuat.
UPDATE participants p
SET portal_user_id = pu.id
FROM portal_users pu
WHERE p.phone = pu.phone
  AND p.portal_user_id IS NULL
  AND p.deleted_at IS NULL;
