-- Migration: Add M-PESA STK Push columns to orders for existing databases.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS mpesa_checkout_request_id VARCHAR(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS mpesa_merchant_request_id VARCHAR(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS mpesa_receipt_number VARCHAR(50);

CREATE INDEX IF NOT EXISTS idx_orders_mpesa_checkout ON orders(mpesa_checkout_request_id) WHERE mpesa_checkout_request_id IS NOT NULL;
