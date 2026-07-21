ALTER TABLE telegramusers
  ADD COLUMN IF NOT EXISTS last_business_connection_id_hash TEXT DEFAULT NULL;