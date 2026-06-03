-- ============================================================
-- Migration 004: Soft Delete + Fix Index + Minor Corrections
-- Per PRD §13.1: "gunakan kolom deleted_at — jangan hapus fisik"
-- ============================================================

-- Fix broken index from migration 003 (referenced non-existent deleted_at)
DROP INDEX IF EXISTS idx_users_email;

-- Add deleted_at to users table, then recreate correct index
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE UNIQUE INDEX idx_users_email_active ON users(email) WHERE deleted_at IS NULL;

-- Add deleted_at to transactional tables per §13.1
ALTER TABLE packages         ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE package_batches  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE leads            ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE participants     ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE invoices         ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE documents        ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE package_images   ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Update existing indexes to exclude soft-deleted rows
DROP INDEX IF EXISTS idx_packages_slug;
CREATE UNIQUE INDEX idx_packages_slug ON packages(slug) WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_packages_active_cat;
CREATE INDEX idx_packages_active_cat ON packages(is_active, category) WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_leads_status_assigned;
CREATE INDEX idx_leads_status_assigned ON leads(status, assigned_to) WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_leads_phone;
CREATE INDEX idx_leads_phone ON leads(phone) WHERE deleted_at IS NULL;

-- Add wa_group_link to package_batches (referenced in PRD §6.1 business flow)
ALTER TABLE package_batches ADD COLUMN IF NOT EXISTS wa_group_link TEXT;

-- Add filter support for departure month (batch)
-- (no schema change needed — use departure_date column directly)

-- Add consent_given to leads (§25.2 informed consent)
ALTER TABLE leads ADD COLUMN IF NOT EXISTS consent_given BOOLEAN NOT NULL DEFAULT FALSE;
