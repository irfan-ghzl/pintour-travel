-- ============================================================
-- Migration 002: payments, participant_documents, booking columns
-- ============================================================

-- ── Feature A: payments ───────────────────────────────────────────────────────
CREATE TABLE payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id      UUID            NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    payment_type    VARCHAR(50)     NOT NULL DEFAULT 'dp',  -- 'dp', 'pelunasan', 'full'
    amount          NUMERIC(15, 2)  NOT NULL,
    paid_at         TIMESTAMPTZ     NOT NULL,
    proof_url       TEXT,
    notes           TEXT,
    verified_by     UUID            REFERENCES users(id) ON DELETE SET NULL,
    verified_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_booking ON payments(booking_id);

-- ── Feature B: participant_documents ─────────────────────────────────────────
CREATE TABLE participant_documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id  UUID            NOT NULL REFERENCES booking_participants(id) ON DELETE CASCADE,
    doc_type        VARCHAR(100)    NOT NULL,  -- 'passport', 'ktp', 'bank_statement', 'visa_support'
    file_url        TEXT            NOT NULL,
    notes           TEXT,
    verified        BOOLEAN         NOT NULL DEFAULT FALSE,
    verified_by     UUID            REFERENCES users(id) ON DELETE SET NULL,
    verified_at     TIMESTAMPTZ,
    uploaded_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_participant_docs_participant ON participant_documents(participant_id);

-- ── Feature C: booking new columns ───────────────────────────────────────────
ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS tour_leader_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS wa_group_link   TEXT,
    ADD COLUMN IF NOT EXISTS briefing_done   BOOLEAN NOT NULL DEFAULT FALSE;
