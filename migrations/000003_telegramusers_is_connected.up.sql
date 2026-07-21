ALTER TABLE telegramusers
  ADD COLUMN IF NOT EXISTS is_connected BOOLEAN NOT NULL DEFAULT false;

UPDATE telegramusers
SET is_connected = true
WHERE business_connection_id_hash IS NOT NULL
  AND business_connection_id_hash <> '';

ALTER TABLE telegramusers
  DROP COLUMN IF EXISTS last_business_connection_id_hash;
