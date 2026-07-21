-- Restore bytea FK columns from id_hash (best-effort; rows without a matching user are dropped).

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
      AND column_name = 'first_user_id_hash'
  ) THEN
    RETURN;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'user_relations'
      AND column_name = 'first_user_id'
  ) THEN
    RETURN;
  END IF;

  ALTER TABLE user_relations
    ADD COLUMN first_user_id BYTEA,
    ADD COLUMN second_user_id BYTEA;

  UPDATE user_relations AS ur
  SET first_user_id = tu.id
  FROM telegramusers AS tu
  WHERE ur.first_user_id_hash = tu.id_hash;

  UPDATE user_relations AS ur
  SET second_user_id = tu.id
  FROM telegramusers AS tu
  WHERE ur.second_user_id_hash = tu.id_hash;

  DELETE FROM user_relations
  WHERE first_user_id IS NULL
     OR second_user_id IS NULL;

  DROP INDEX IF EXISTS idx_relations_first_user;
  DROP INDEX IF EXISTS idx_relations_second_user;

  ALTER TABLE user_relations DROP COLUMN IF EXISTS first_user_id_hash CASCADE;
  ALTER TABLE user_relations DROP COLUMN IF EXISTS second_user_id_hash CASCADE;

  ALTER TABLE user_relations
    ALTER COLUMN first_user_id SET NOT NULL,
    ALTER COLUMN second_user_id SET NOT NULL;

  CREATE INDEX IF NOT EXISTS idx_relations_first_user
    ON user_relations (first_user_id);

  CREATE INDEX IF NOT EXISTS idx_relations_second_user
    ON user_relations (second_user_id);
END $$;
