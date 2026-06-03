-- ============================================================
-- Migration 003: PRD-Compliant Schema Refactor
-- Replaces old inquiry/quotation/booking model with
-- leads → participants → invoice → portal flow per PRD v1.5
-- ============================================================

-- Drop old tables (order matters due to FK constraints)
DROP TABLE IF EXISTS booking_participants CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS participant_documents CASCADE;
DROP TABLE IF EXISTS bookings CASCADE;
DROP TABLE IF EXISTS quotation_items CASCADE;
DROP TABLE IF EXISTS quotations CASCADE;
DROP TABLE IF EXISTS inquiries CASCADE;
DROP TABLE IF EXISTS package_galleries CASCADE;
DROP TABLE IF EXISTS itinerary_items CASCADE;
DROP TABLE IF EXISTS testimonials CASCADE;
DROP TABLE IF EXISTS tour_packages CASCADE;
DROP TABLE IF EXISTS destinations CASCADE;

-- Update users table: add phone, update role constraint
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20);
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('super_admin', 'admin', 'konsultan', 'tour_leader'));

UPDATE users SET role = 'admin' WHERE role NOT IN ('super_admin', 'admin', 'konsultan', 'tour_leader');

-- ============================================================
-- Table: packages
-- ============================================================
CREATE TABLE packages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(200)        NOT NULL,
    slug            VARCHAR(200) UNIQUE NOT NULL,
    destination     VARCHAR(100)        NOT NULL,
    category        VARCHAR(20)         NOT NULL DEFAULT 'reguler'
                        CHECK (category IN ('reguler', 'umroh', 'halal', 'honeymoon')),
    duration_days   INTEGER             NOT NULL CHECK (duration_days BETWEEN 1 AND 365),
    description     TEXT,
    itinerary       JSONB               NOT NULL DEFAULT '[]',
    requirements    JSONB               NOT NULL DEFAULT '[]',
    base_price      NUMERIC(12,2)       NOT NULL CHECK (base_price > 0),
    is_active       BOOLEAN             NOT NULL DEFAULT TRUE,
    thumbnail       VARCHAR(500),
    created_by      UUID                REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: package_images
-- ============================================================
CREATE TABLE package_images (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package_id  UUID            NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    file_path   VARCHAR(500)    NOT NULL,
    alt_text    VARCHAR(200),
    sort_order  INTEGER         NOT NULL DEFAULT 0,
    is_thumbnail BOOLEAN        NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: tour_leaders
-- ============================================================
CREATE TABLE tour_leaders (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID UNIQUE     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bio                 TEXT,
    photo_path          VARCHAR(500),
    experience_years    INTEGER         NOT NULL DEFAULT 0,
    specialization      VARCHAR(200),
    emergency_phone     VARCHAR(20),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: package_batches
-- ============================================================
CREATE TABLE package_batches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package_id      UUID            NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    departure_date  DATE            NOT NULL,
    return_date     DATE            NOT NULL,
    quota           INTEGER         NOT NULL CHECK (quota > 0),
    price_single    NUMERIC(12,2)   NOT NULL CHECK (price_single > 0),
    price_double    NUMERIC(12,2)   NOT NULL CHECK (price_double > 0),
    price_triple    NUMERIC(12,2)   NOT NULL CHECK (price_triple > 0),
    status          VARCHAR(20)     NOT NULL DEFAULT 'tersedia'
                        CHECK (status IN ('tersedia', 'penuh', 'ditutup')),
    tour_leader_id  UUID            REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: leads
-- ============================================================
CREATE TABLE leads (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100)    NOT NULL,
    phone       VARCHAR(20)     NOT NULL,
    email       VARCHAR(100),
    package_id  UUID            NOT NULL REFERENCES packages(id) ON DELETE RESTRICT,
    batch_id    UUID            REFERENCES package_batches(id) ON DELETE SET NULL,
    pax         INTEGER         NOT NULL DEFAULT 1 CHECK (pax BETWEEN 1 AND 50),
    message     TEXT,
    source      VARCHAR(50)     NOT NULL DEFAULT 'organic'
                    CHECK (source IN ('meta_ads', 'organic', 'referral', 'walk_in')),
    status      VARCHAR(20)     NOT NULL DEFAULT 'baru'
                    CHECK (status IN ('baru', 'dihubungi', 'konsultasi', 'deal', 'tidak_deal', 'peserta')),
    assigned_to UUID            REFERENCES users(id) ON DELETE SET NULL,
    converted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: lead_notes
-- ============================================================
CREATE TABLE lead_notes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id     UUID            NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    user_id     UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    note        TEXT            NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: participants
-- ============================================================
CREATE TABLE participants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id         UUID            REFERENCES leads(id) ON DELETE SET NULL,
    batch_id        UUID            NOT NULL REFERENCES package_batches(id) ON DELETE RESTRICT,
    name            VARCHAR(100)    NOT NULL,
    phone           VARCHAR(20)     NOT NULL,
    email           VARCHAR(100),
    room_type       VARCHAR(10)     NOT NULL DEFAULT 'double'
                        CHECK (room_type IN ('single', 'double', 'triple')),
    portal_password VARCHAR(255)    NOT NULL,
    is_active       BOOLEAN         NOT NULL DEFAULT FALSE,
    briefing_viewed BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: invoices
-- ============================================================
CREATE TABLE invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number  VARCHAR(30) UNIQUE NOT NULL,
    participant_id  UUID            NOT NULL REFERENCES participants(id) ON DELETE RESTRICT,
    batch_id        UUID            NOT NULL REFERENCES package_batches(id) ON DELETE RESTRICT,
    amount          NUMERIC(12,2)   NOT NULL CHECK (amount > 0),
    due_date        DATE            NOT NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'diterbitkan'
                        CHECK (status IN ('diterbitkan', 'menunggu_bayar', 'dibayar', 'lunas')),
    pdf_path        VARCHAR(500),
    notes           TEXT,
    issued_by       UUID            NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    confirmed_by    UUID            REFERENCES users(id) ON DELETE SET NULL,
    confirmed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: payment_proofs
-- ============================================================
CREATE TABLE payment_proofs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id      UUID            NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    file_path       VARCHAR(500)    NOT NULL,
    amount_claimed  NUMERIC(12,2)   NOT NULL CHECK (amount_claimed > 0),
    notes           TEXT,
    status          VARCHAR(20)     NOT NULL DEFAULT 'menunggu'
                        CHECK (status IN ('menunggu', 'disetujui', 'ditolak')),
    reviewed_by     UUID            REFERENCES users(id) ON DELETE SET NULL,
    review_notes    TEXT,
    uploaded_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    reviewed_at     TIMESTAMPTZ
);

-- ============================================================
-- Table: country_document_requirements
-- ============================================================
CREATE TABLE country_document_requirements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_code    VARCHAR(5)      NOT NULL,
    country_name    VARCHAR(100)    NOT NULL,
    document_type   VARCHAR(50)     NOT NULL,
    is_required     BOOLEAN         NOT NULL DEFAULT TRUE,
    description     TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: documents
-- ============================================================
CREATE TABLE documents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id      UUID            NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    document_type       VARCHAR(50)     NOT NULL
                            CHECK (document_type IN ('passport', 'ktp', 'rekening_koran', 'visa_support', 'lainnya')),
    file_path           VARCHAR(500)    NOT NULL,
    file_name           VARCHAR(200)    NOT NULL,
    status              VARCHAR(20)     NOT NULL DEFAULT 'menunggu'
                            CHECK (status IN ('menunggu', 'disetujui', 'ditolak')),
    rejection_reason    TEXT,
    reviewed_by         UUID            REFERENCES users(id) ON DELETE SET NULL,
    uploaded_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    reviewed_at         TIMESTAMPTZ
);

-- ============================================================
-- Table: airport_checklists
-- ============================================================
CREATE TABLE airport_checklists (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id          UUID            NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    batch_id                UUID            NOT NULL REFERENCES package_batches(id) ON DELETE CASCADE,
    baggage_checked         BOOLEAN         NOT NULL DEFAULT FALSE,
    baggage_checked_at      TIMESTAMPTZ,
    ticket_distributed      BOOLEAN         NOT NULL DEFAULT FALSE,
    ticket_distributed_at   TIMESTAMPTZ,
    passport_returned       BOOLEAN         NOT NULL DEFAULT FALSE,
    passport_returned_at    TIMESTAMPTZ,
    handled_by              UUID            REFERENCES users(id) ON DELETE SET NULL,
    notes                   TEXT,
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (participant_id, batch_id)
);

-- ============================================================
-- Table: wa_notifications
-- ============================================================
CREATE TABLE wa_notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_phone VARCHAR(20)     NOT NULL,
    recipient_name  VARCHAR(100)    NOT NULL,
    message_type    VARCHAR(50)     NOT NULL,
    message_content TEXT            NOT NULL,
    reference_id    UUID,
    reference_type  VARCHAR(50),
    status          VARCHAR(20)     NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'sent', 'failed')),
    fonnte_response JSONB,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Indexes
-- ============================================================
-- idx_users_email akan dibuat ulang dengan filter deleted_at di migration 004
-- (saat itu kolom deleted_at sudah ada). Sementara cukup pakai constraint UNIQUE
-- yang sudah ada di tabel users (dari migration 001).
-- CREATE UNIQUE INDEX idx_users_email ON users(email);
CREATE UNIQUE INDEX idx_packages_slug ON packages(slug);
CREATE INDEX idx_packages_active_cat ON packages(is_active, category);
CREATE INDEX idx_package_batches_pkg_date ON package_batches(package_id, departure_date);
CREATE INDEX idx_leads_status_assigned ON leads(status, assigned_to);
CREATE INDEX idx_leads_phone ON leads(phone);
CREATE UNIQUE INDEX idx_invoices_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_participant_status ON invoices(participant_id, status);
CREATE INDEX idx_documents_participant_status ON documents(participant_id, status);
CREATE INDEX idx_wa_notif_ref ON wa_notifications(reference_id, reference_type);
CREATE INDEX idx_airport_batch ON airport_checklists(batch_id);
