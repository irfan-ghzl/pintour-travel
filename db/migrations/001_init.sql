-- ============================================================
-- Schema: pintour-travel database
-- ============================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- Table: users
-- ============================================================
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255)        NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    password    TEXT                NOT NULL,
    role        VARCHAR(50)         NOT NULL DEFAULT 'staff',
    is_active   BOOLEAN             NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: destinations
-- ============================================================
CREATE TABLE destinations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255)  NOT NULL,
    country     VARCHAR(100)  NOT NULL,
    description TEXT,
    image_url   TEXT,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: tour_packages
-- ============================================================
CREATE TABLE tour_packages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    destination_id  UUID            REFERENCES destinations(id) ON DELETE SET NULL,
    title           VARCHAR(255)    NOT NULL,
    slug            VARCHAR(255)    UNIQUE NOT NULL,
    description     TEXT,
    price           NUMERIC(15, 2)  NOT NULL DEFAULT 0,
    price_label     VARCHAR(100),
    duration_days   INTEGER         NOT NULL DEFAULT 1,
    max_participants INTEGER,
    min_participants INTEGER         NOT NULL DEFAULT 1,
    package_type    VARCHAR(50)     NOT NULL DEFAULT 'regular',
    cover_image_url TEXT,
    is_active       BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: itinerary_items
-- ============================================================
CREATE TABLE itinerary_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tour_package_id UUID            NOT NULL REFERENCES tour_packages(id) ON DELETE CASCADE,
    day_number      INTEGER         NOT NULL,
    title           VARCHAR(255)    NOT NULL,
    description     TEXT,
    location        VARCHAR(255),
    start_time      VARCHAR(10),
    end_time        VARCHAR(10),
    activity_type   VARCHAR(100),
    sort_order      INTEGER         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: package_galleries
-- ============================================================
CREATE TABLE package_galleries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tour_package_id UUID        NOT NULL REFERENCES tour_packages(id) ON DELETE CASCADE,
    image_url       TEXT        NOT NULL,
    caption         VARCHAR(255),
    sort_order      INTEGER     NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: inquiries (leads)
-- ============================================================
CREATE TABLE inquiries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name       VARCHAR(255)    NOT NULL,
    email           VARCHAR(255),
    phone           VARCHAR(50),
    destination     VARCHAR(255),
    tour_package_id UUID            REFERENCES tour_packages(id) ON DELETE SET NULL,
    num_people      INTEGER         NOT NULL DEFAULT 1,
    budget          NUMERIC(15, 2),
    duration_days   INTEGER,
    departure_date  DATE,
    notes           TEXT,
    status          VARCHAR(50)     NOT NULL DEFAULT 'new',
    assigned_to     UUID            REFERENCES users(id) ON DELETE SET NULL,
    wa_link         TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: quotations
-- ============================================================
CREATE TABLE quotations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inquiry_id      UUID            REFERENCES inquiries(id) ON DELETE SET NULL,
    created_by      UUID            REFERENCES users(id) ON DELETE SET NULL,
    title           VARCHAR(255)    NOT NULL,
    customer_name   VARCHAR(255)    NOT NULL,
    customer_email  VARCHAR(255),
    customer_phone  VARCHAR(50),
    valid_until     DATE,
    total_price     NUMERIC(15, 2)  NOT NULL DEFAULT 0,
    notes           TEXT,
    status          VARCHAR(50)     NOT NULL DEFAULT 'draft',
    pdf_url         TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: quotation_items
-- ============================================================
CREATE TABLE quotation_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quotation_id    UUID            NOT NULL REFERENCES quotations(id) ON DELETE CASCADE,
    description     VARCHAR(255)    NOT NULL,
    category        VARCHAR(100),
    quantity        INTEGER         NOT NULL DEFAULT 1,
    unit_price      NUMERIC(15, 2)  NOT NULL DEFAULT 0,
    total_price     NUMERIC(15, 2)  GENERATED ALWAYS AS (quantity * unit_price) STORED,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: bookings
-- ============================================================
CREATE TABLE bookings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tour_package_id UUID            REFERENCES tour_packages(id) ON DELETE SET NULL,
    quotation_id    UUID            REFERENCES quotations(id) ON DELETE SET NULL,
    booking_code    VARCHAR(50)     UNIQUE NOT NULL,
    customer_name   VARCHAR(255)    NOT NULL,
    customer_email  VARCHAR(255),
    customer_phone  VARCHAR(50),
    departure_date  DATE            NOT NULL,
    num_people      INTEGER         NOT NULL DEFAULT 1,
    total_price     NUMERIC(15, 2)  NOT NULL DEFAULT 0,
    payment_status  VARCHAR(50)     NOT NULL DEFAULT 'pending',
    booking_status  VARCHAR(50)     NOT NULL DEFAULT 'confirmed',
    notes           TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: booking_participants
-- ============================================================
CREATE TABLE booking_participants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id      UUID            NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    full_name       VARCHAR(255)    NOT NULL,
    id_type         VARCHAR(20)     NOT NULL DEFAULT 'ktp',
    id_number       VARCHAR(50)     NOT NULL,
    date_of_birth   DATE,
    phone           VARCHAR(50),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: testimonials
-- ============================================================
CREATE TABLE testimonials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tour_package_id UUID            REFERENCES tour_packages(id) ON DELETE SET NULL,
    customer_name   VARCHAR(255)    NOT NULL,
    content         TEXT            NOT NULL,
    rating          SMALLINT        NOT NULL DEFAULT 5,
    photo_url       TEXT,
    is_published    BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Indexes
-- ============================================================
CREATE INDEX idx_tour_packages_destination ON tour_packages(destination_id);
CREATE INDEX idx_tour_packages_slug ON tour_packages(slug);
CREATE INDEX idx_tour_packages_active ON tour_packages(is_active);
CREATE INDEX idx_itinerary_items_package ON itinerary_items(tour_package_id, day_number);
CREATE INDEX idx_inquiries_status ON inquiries(status);
CREATE INDEX idx_inquiries_created ON inquiries(created_at DESC);
CREATE INDEX idx_quotations_inquiry ON quotations(inquiry_id);
CREATE INDEX idx_bookings_package ON bookings(tour_package_id);
CREATE INDEX idx_bookings_departure ON bookings(departure_date);
