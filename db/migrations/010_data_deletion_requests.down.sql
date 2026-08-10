-- ============================================================
-- Rollback 010: Permintaan penghapusan data
-- ============================================================
--
-- What this loses:
--
--   * data_deletion_requests is dropped, and with it every erasure request that
--     has been received but not yet processed. Those are legal obligations with
--     a clock running on them (UU PDP Pasal 46, 14 working days) and they cannot
--     be reconstructed — the participant is the only other party who knows they
--     asked. Export the table before rolling back if any row is still 'menunggu'.
--   * participants.anonymized_at is dropped. Participants that were already
--     anonymised STAY anonymised — their name, phone, and email are overwritten,
--     not hidden — but the record of when that happened is gone.
--
-- Take a backup first.

DROP INDEX IF EXISTS idx_deletion_requests_status;
DROP INDEX IF EXISTS idx_deletion_requests_open;
DROP TABLE IF EXISTS data_deletion_requests;

ALTER TABLE participants DROP COLUMN IF EXISTS anonymized_at;
