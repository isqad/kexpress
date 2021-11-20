BEGIN;
ALTER TABLE products RENAME COLUMN category_id TO portal_category_id;
ALTER TABLE products ADD COLUMN category_id bigint;

UPDATE products SET category_id = categories.id FROM categories WHERE categories.portal_id = portal_category_id;
ALTER TABLE products ALTER COLUMN category_id SET NOT NULL;
ALTER TABLE products ADD CONSTRAINT fk_products_categories FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE;
CREATE INDEX index_products_category_id ON products (category_id);
COMMIT;
