ALTER TABLE categories ADD COLUMN history jsonb NOT NULL DEFAULT '{}'::jsonb;
