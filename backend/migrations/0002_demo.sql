-- Демо-данные: чтобы в кадре не было пустых экранов.
-- Логины: admin@shop.test / admin12345, garden@shop.test и home@shop.test / seller12345
CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO users (email, password_hash, role_id) VALUES
 ('admin@shop.test',  crypt('admin12345',  gen_salt('bf', 10)), (SELECT id FROM roles WHERE name = 'admin')),
 ('garden@shop.test', crypt('seller12345', gen_salt('bf', 10)), (SELECT id FROM roles WHERE name = 'seller')),
 ('home@shop.test',   crypt('seller12345', gen_salt('bf', 10)), (SELECT id FROM roles WHERE name = 'seller'));

INSERT INTO shops (owner_id, name, logo_url, description, contacts, status) VALUES
 ((SELECT id FROM users WHERE email = 'garden@shop.test'), 'Садовый уголок',
  'https://placehold.co/200x200?text=Garden', 'Всё для дачи и огорода: инструменты, семена, перчатки.', '+7 900 111-22-33, garden@shop.test', 'approved'),
 ((SELECT id FROM users WHERE email = 'home@shop.test'), 'Уютный дом',
  'https://placehold.co/200x200?text=Home', 'Тёплые мелочи для дома: пледы, свечи, декор.', '+7 900 444-55-66, home@shop.test', 'approved');

INSERT INTO products (shop_id, category_id, name, description, price, stock, status) VALUES
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), (SELECT id FROM categories WHERE name = 'Садовые инструменты'),
  'Секатор профессиональный', 'Прочный секатор с тефлоновым покрытием лезвий и удобной эргономичной ручкой.', 890, 25, 'published'),
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), (SELECT id FROM categories WHERE name = 'Садовые инструменты'),
  'Лейка 5 литров', 'Пластиковая лейка с длинным носиком и съёмной душевой насадкой.', 450, 40, 'published'),
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), (SELECT id FROM categories WHERE name = 'Семена и растения'),
  'Набор семян томатов', 'Пять сортов томатов для открытого грунта и теплицы.', 190, 100, 'published'),
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), (SELECT id FROM categories WHERE name = 'Садовые инструменты'),
  'Перчатки садовые', 'Плотные перчатки с нитриловым покрытием ладони.', 320, 3, 'published'),
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), (SELECT id FROM categories WHERE name = 'Декор для дома'),
  'Горшок керамический', 'Керамический горшок 2 литра с поддоном.', 650, 0, 'published'),
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), (SELECT id FROM categories WHERE name = 'Подарки'),
  'Подарочный набор садовода', 'Секатор, перчатки и семена в подарочной коробке.', 2490, 10, 'pending'),
 ((SELECT id FROM shops WHERE name = 'Уютный дом'), (SELECT id FROM categories WHERE name = 'Декор для дома'),
  'Плед флисовый', 'Мягкий флисовый плед 150×200 см, не линяет при стирке.', 1490, 15, 'published'),
 ((SELECT id FROM shops WHERE name = 'Уютный дом'), (SELECT id FROM categories WHERE name = 'Декор для дома'),
  'Ваза стеклянная', 'Прозрачная ваза высотой 25 см.', 990, 8, 'published'),
 ((SELECT id FROM shops WHERE name = 'Уютный дом'), (SELECT id FROM categories WHERE name = 'Подарки'),
  'Свеча ароматическая', 'Соевая свеча с ароматом ванили, время горения около 30 часов.', 350, 60, 'published');

INSERT INTO product_images (product_id, url, sort_order)
SELECT id, 'https://placehold.co/600x600?text=Product+' || id, 0 FROM products;

INSERT INTO product_attributes (product_id, name, value)
SELECT p.id, v.name, v.value FROM (VALUES
 ('Секатор профессиональный', 'Материал лезвия', 'сталь'),
 ('Секатор профессиональный', 'Длина', '21 см'),
 ('Лейка 5 литров', 'Объём', '5 л'),
 ('Плед флисовый', 'Размер', '150×200 см'),
 ('Плед флисовый', 'Материал', 'флис'),
 ('Свеча ароматическая', 'Аромат', 'ваниль')
) AS v(pname, name, value) JOIN products p ON p.name = v.pname;

INSERT INTO orders (shop_id, number, customer_name, customer_phone, customer_email, delivery_address, total_amount, status, payment_status, created_at) VALUES
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), 'ORD-DEMO-0001', 'Анна Смирнова', '+79001112233', 'anna@example.com', 'Екатеринбург, ул. Ленина, 1', 1340, 'delivered',  'paid',   now() - interval '12 days'),
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), 'ORD-DEMO-0002', 'Иван Петров',   '+79002223344', 'ivan@example.com', 'Екатеринбург, ул. Мира, 5',   1270, 'shipped',    'paid',   now() - interval '6 days'),
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), 'ORD-DEMO-0003', 'Ольга Кузнецова','+79003334455', 'olga@example.com', 'Челябинск, пр. Победы, 10',  1780, 'new',        'unpaid', now() - interval '1 day'),
 ((SELECT id FROM shops WHERE name = 'Садовый уголок'), 'ORD-DEMO-0004', 'Пётр Сидоров',  '+79004445566', 'petr@example.com', 'Пермь, ул. Садовая, 3',       450,  'cancelled',  'unpaid', now() - interval '9 days'),
 ((SELECT id FROM shops WHERE name = 'Уютный дом'),     'ORD-DEMO-0005', 'Мария Орлова',  '+79005556677', 'maria@example.com','Екатеринбург, ул. Репина, 7',  2190, 'delivered',  'paid',   now() - interval '10 days'),
 ((SELECT id FROM shops WHERE name = 'Уютный дом'),     'ORD-DEMO-0006', 'Дмитрий Волков','+79006667788', 'dmitry@example.com','Тюмень, ул. Республики, 12', 990,  'assembling', 'paid',   now() - interval '3 days'),
 ((SELECT id FROM shops WHERE name = 'Уютный дом'),     'ORD-DEMO-0007', 'Елена Новикова','+79007778899', 'elena@example.com','Екатеринбург, ул. Ленина, 30',1050, 'delivered',  'paid',   now() - interval '2 days');

INSERT INTO order_items (order_id, product_id, product_name, quantity, unit_price)
SELECT o.id, p.id, p.name, v.qty, p.price FROM (VALUES
 ('ORD-DEMO-0001', 'Секатор профессиональный', 1),
 ('ORD-DEMO-0001', 'Лейка 5 литров', 1),
 ('ORD-DEMO-0002', 'Набор семян томатов', 5),
 ('ORD-DEMO-0002', 'Перчатки садовые', 1),
 ('ORD-DEMO-0003', 'Секатор профессиональный', 2),
 ('ORD-DEMO-0004', 'Лейка 5 литров', 1),
 ('ORD-DEMO-0005', 'Плед флисовый', 1),
 ('ORD-DEMO-0005', 'Свеча ароматическая', 2),
 ('ORD-DEMO-0006', 'Ваза стеклянная', 1),
 ('ORD-DEMO-0007', 'Свеча ароматическая', 3)
) AS v(num, pname, qty)
JOIN orders o ON o.number = v.num
JOIN products p ON p.name = v.pname;

INSERT INTO order_status_history (order_id, status, changed_by, changed_at)
SELECT id, 'new', NULL, created_at FROM orders;
INSERT INTO order_status_history (order_id, status, changed_by, changed_at)
SELECT o.id, o.status, (SELECT owner_id FROM shops WHERE id = o.shop_id), o.created_at + interval '1 day'
FROM orders o WHERE o.status <> 'new';

INSERT INTO payments (order_id, amount, status, created_at)
SELECT id, total_amount, 'success', created_at FROM orders WHERE payment_status = 'paid';

INSERT INTO moderation_records (shop_id, moderator_id, decision, reason)
SELECT s.id, (SELECT id FROM users WHERE email = 'admin@shop.test'), 'approved', '' FROM shops s;
