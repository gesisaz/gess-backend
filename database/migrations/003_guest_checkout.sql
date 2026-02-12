-- Migration: Guest checkout - nullable user_id/shipping_address_id and inline guest shipping.

ALTER TABLE orders ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE orders ALTER COLUMN shipping_address_id DROP NOT NULL;

ALTER TABLE orders ADD COLUMN IF NOT EXISTS guest_email VARCHAR(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS guest_name VARCHAR(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_full_name VARCHAR(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_street_address TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_city VARCHAR(100);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_state VARCHAR(100);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_postal_code VARCHAR(20);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_country VARCHAR(100);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_phone VARCHAR(20);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'orders_owner_check'
      AND conrelid = 'orders'::regclass
  ) THEN
    ALTER TABLE orders ADD CONSTRAINT orders_owner_check CHECK (
      (user_id IS NOT NULL) OR (guest_email IS NOT NULL)
    );
  END IF;
END
$$;
