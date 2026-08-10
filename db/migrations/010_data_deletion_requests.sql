-- ============================================================
-- Migration 010: Permintaan penghapusan data (§25.5 Right to Erasure)
-- ============================================================
--
-- Endpoint permintaan penghapusan sudah ada sejak lama dan menjawab peserta
-- dengan "permintaan Anda akan diproses dalam 14 hari kerja sesuai UU PDP
-- Pasal 46" — tetapi tidak menyimpan apa pun. Tidak ada yang bisa memprosesnya
-- karena tidak ada yang tahu permintaan itu pernah masuk.
--
-- Tabel ini yang membuat janji tersebut punya isi.

CREATE TABLE IF NOT EXISTS data_deletion_requests (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id UUID        NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    reason         TEXT,
    status         VARCHAR(20) NOT NULL DEFAULT 'menunggu'
                       CHECK (status IN ('menunggu', 'selesai', 'ditolak')),
    requested_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_by   UUID        REFERENCES users(id) ON DELETE SET NULL,
    processed_at   TIMESTAMPTZ,
    notes          TEXT
);

-- Satu permintaan terbuka per peserta. Menekan tombolnya dua kali adalah hal
-- yang wajar dilakukan orang yang cemas datanya belum terhapus; itu tidak boleh
-- menghasilkan dua tiket yang harus diproses admin dua kali.
CREATE UNIQUE INDEX IF NOT EXISTS idx_deletion_requests_open
    ON data_deletion_requests(participant_id) WHERE status = 'menunggu';

CREATE INDEX IF NOT EXISTS idx_deletion_requests_status
    ON data_deletion_requests(status, requested_at);

-- Penanda anonimisasi pada peserta (§25.4: data peserta dianonimkan, bukan
-- dihapus — angka statistik keberangkatan harus tetap utuh).
ALTER TABLE participants ADD COLUMN IF NOT EXISTS anonymized_at TIMESTAMPTZ;
