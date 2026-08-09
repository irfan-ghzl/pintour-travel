-- ============================================================
-- Rollback 008: Audit trail & soft delete
-- ============================================================
--
-- What this loses, stated plainly so the choice is made knowingly:
--
--   * lead_status_history is dropped, and with it every record of who moved a
--     lead to which status. That history cannot be reconstructed from the leads
--     table, which keeps only the current status.
--   * payment_proofs.deleted_at and airport_checklists.deleted_at are dropped,
--     so rows that were soft-deleted become visible again. Nothing is destroyed;
--     the marks saying "treat these as gone" are.
--
-- Take a backup first. This is a way out of a bad deploy, not routine.

DROP INDEX IF EXISTS idx_airport_checklists_batch_active;
DROP INDEX IF EXISTS idx_payment_proofs_invoice_active;

ALTER TABLE airport_checklists DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE payment_proofs     DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_lead_status_history_lead;
DROP TABLE IF EXISTS lead_status_history;
