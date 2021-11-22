ALTER TABLE products ADD COLUMN session_id bigint NOT NULL;
ALTER TABLE products ADD CONSTRAINT uniq_portal_id_session_id_products UNIQUE
  (portal_id, session_id);
