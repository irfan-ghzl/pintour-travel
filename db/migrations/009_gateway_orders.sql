-- ============================================================
-- Migration 009: Payment gateway accuracy (v2.0 F1)
--   Cacat 1: notifikasi berulang membuat bukti bayar berulang
--   Cacat 2: pengenal order ditimpa setiap sesi pembayaran baru
-- ============================================================

-- ── Order gateway: satu invoice, banyak sesi ────────────────────────────────
-- Sebelumnya pengenal order disimpan pada invoices.midtrans_order_id dan
-- DITIMPA setiap kali peserta membuka halaman pembayaran lagi. Peserta yang
-- memuat ulang lalu menyelesaikan pembayaran di tab pertama mengirim notifikasi
-- dengan order lama — yang sudah tidak dikenali sistem, sehingga uangnya
-- diterima gateway tapi tidak pernah tercatat.
--
-- Relasinya kini satu-ke-banyak: setiap sesi Snap menambah baris, tidak ada
-- yang hilang, dan notifikasi atas sesi manapun tetap ketemu invoicenya.
CREATE TABLE IF NOT EXISTS invoice_gateway_orders (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID         NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    order_id   VARCHAR(100) NOT NULL UNIQUE,
    snap_token TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoice_gateway_orders_invoice
    ON invoice_gateway_orders(invoice_id, created_at DESC);

-- Baris yang sudah ada dipindahkan supaya sesi yang dibuat sebelum migrasi ini
-- tetap dapat dicocokkan.
INSERT INTO invoice_gateway_orders (invoice_id, order_id, snap_token)
SELECT id, midtrans_order_id, snap_token
FROM invoices
WHERE midtrans_order_id IS NOT NULL AND midtrans_order_id <> ''
ON CONFLICT (order_id) DO NOTHING;

-- ── Idempotensi notifikasi gateway ──────────────────────────────────────────
-- Pemrosesan dulu hanya berhenti lebih awal bila invoice sudah lunas. Untuk
-- pembayaran sebagian invoice masih 'menunggu_bayar', jadi setiap pengiriman
-- ulang notifikasi membuat bukti bayar baru bernominal sama — empat kali kirim
-- ulang atas satu uang muka bisa menyatakan invoice lunas tanpa uang tambahan.
--
-- Kuncinya adalah identitas transaksi, bukan status invoice: satu baris per
-- notifikasi yang sudah diterapkan, dan UNIQUE-nya yang menolak yang kedua.
CREATE TABLE IF NOT EXISTS gateway_notifications (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key    VARCHAR(200) NOT NULL UNIQUE,
    invoice_id         UUID         NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    order_id           VARCHAR(100) NOT NULL,
    transaction_id     VARCHAR(100),
    transaction_status VARCHAR(30)  NOT NULL,
    applied_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gateway_notifications_invoice
    ON gateway_notifications(invoice_id);
