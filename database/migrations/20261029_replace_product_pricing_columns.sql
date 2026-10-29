USE go_pulsa_db;

ALTER TABLE products
    ADD COLUMN harga_jual DECIMAL(15,2) NOT NULL DEFAULT 0.00 AFTER price;

UPDATE products
SET harga_jual = GREATEST(price, base_price + default_markup);

UPDATE transactions AS t
JOIN products AS p ON p.id = t.product_id
SET t.amount = p.nominal,
    t.sell_price = p.harga_jual,
    t.markup = p.harga_jual - p.price
WHERE t.status = 'PENDING'
  AND (t.amount <= 0 OR t.sell_price <= 0);

ALTER TABLE products
    DROP COLUMN base_price,
    DROP COLUMN default_markup;
