-- Replace bytea FK columns (first_user_id, second_user_id) with id_hash text columns.
-- No-op when the table is missing or already uses the new column names (fresh bootstrap).

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public'
      AND table_name = 'user_relations'
  ) THEN
    RETURN;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'user_relations'
      AND column_name = 'first_user_id'
  ) THEN
    RETURN;
  END IF;

  ALTER TABLE user_relations
    ADD COLUMN IF NOT EXISTS first_user_id_hash TEXT,
    ADD COLUMN IF NOT EXISTS second_user_id_hash TEXT;

  UPDATE user_relations AS ur
  SET first_user_id_hash = tu.id_hash
  FROM telegramusers AS tu
  WHERE ur.first_user_id = tu.id
    AND ur.first_user_id_hash IS NULL;

  UPDATE user_relations AS ur
  SET second_user_id_hash = tu.id_hash
  FROM telegramusers AS tu
  WHERE ur.second_user_id = tu.id
    AND ur.second_user_id_hash IS NULL;

  DELETE FROM user_relations
  WHERE first_user_id_hash IS NULL
     OR second_user_id_hash IS NULL;

  DROP INDEX IF EXISTS idx_relations_first_user;
  DROP INDEX IF EXISTS idx_relations_second_user;

  ALTER TABLE user_relations DROP COLUMN IF EXISTS first_user_id CASCADE;
  ALTER TABLE user_relations DROP COLUMN IF EXISTS second_user_id CASCADE;

  ALTER TABLE user_relations
    ALTER COLUMN first_user_id_hash SET NOT NULL,
    ALTER COLUMN second_user_id_hash SET NOT NULL;

  CREATE INDEX IF NOT EXISTS idx_relations_first_user
    ON user_relations (first_user_id_hash);

  CREATE INDEX IF NOT EXISTS idx_relations_second_user
    ON user_relations (second_user_id_hash);
END $$;
