ALTER TABLE messages ADD COLUMN IF NOT EXISTS metadata_format smallint NOT NULL DEFAULT 0;
