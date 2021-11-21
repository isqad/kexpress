ALTER TABLE products DROP COLUMN raw_json;

ALTER TABLE products ADD COLUMN fingerprint varchar(1024) NOT NULL DEFAULT '';
CREATE UNIQUE INDEX uniq_products_fingerprint ON products (fingerprint) WHERE fingerprint != '';
