-- ============================================================
-- Rollback 009: Payment gateway accuracy
-- ============================================================
--
-- What this loses:
--
--   * gateway_notifications is dropped, and with it the record of which gateway
--     notifications have already been applied. Idempotency goes with it: after
--     this, a redelivered settlement is indistinguishable from a new payment and
--     will be counted again. That is the defect 009 existed to fix, so rolling
--     back reopens it — reconcile payments by hand until 009 is applied again.
--   * invoice_gateway_orders is dropped. Sessions opened BEFORE 009 survive,
--     because the column they were copied from (invoices.midtrans_order_id) is
--     still there and untouched. Sessions opened AFTER 009 exist only in this
--     table and are lost, so a notification naming one can no longer be matched
--     to its invoice.
--
-- Take a backup first, and prefer fixing forward.

DROP INDEX IF EXISTS idx_gateway_notifications_invoice;
DROP TABLE IF EXISTS gateway_notifications;

DROP INDEX IF EXISTS idx_invoice_gateway_orders_invoice;
DROP TABLE IF EXISTS invoice_gateway_orders;
