-- Схема БД по модели тимлида (13 сущностей) + небольшие доработки (см. комментарии "ДОРАБОТКА")

-- Доступ
CREATE TABLE roles (
    id   SMALLSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
INSERT INTO roles (name) VALUES ('seller'), ('admin');

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role_id       SMALLINT NOT NULL REFERENCES roles(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Магазины: у селлера один магазин (owner_id UNIQUE)
CREATE TABLE shops (
    id          BIGSERIAL PRIMARY KEY,
    owner_id    BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    logo_url    TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    contacts    TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Каталог
CREATE TABLE categories (
    id        BIGSERIAL PRIMARY KEY,
    name      TEXT NOT NULL,
    parent_id BIGINT REFERENCES categories(id) ON DELETE SET NULL
);

CREATE TABLE products (
    id          BIGSERIAL PRIMARY KEY,
    shop_id     BIGINT NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price       NUMERIC(12,2) NOT NULL CHECK (price >= 0),
    stock       INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),      -- остаток
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','published','rejected')),
    is_hidden   BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at  TIMESTAMPTZ,                                         -- «мягкое» удаление
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_products_shop ON products(shop_id) WHERE deleted_at IS NULL;      -- ДОРАБОТКА: индексы
CREATE INDEX idx_products_category ON products(category_id) WHERE deleted_at IS NULL;

CREATE TABLE product_images (
    id         BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_images_product ON product_images(product_id);

CREATE TABLE product_attributes (
    id         BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    value      TEXT NOT NULL
);
CREATE INDEX idx_attributes_product ON product_attributes(product_id);

-- Заказы (покупатель без аккаунта: контакты хранятся в заказе)
CREATE TABLE orders (
    id               BIGSERIAL PRIMARY KEY,
    shop_id          BIGINT NOT NULL REFERENCES shops(id),
    number           TEXT NOT NULL UNIQUE,
    customer_name    TEXT NOT NULL,
    customer_phone   TEXT NOT NULL,
    customer_email   TEXT NOT NULL,
    delivery_address TEXT NOT NULL,
    total_amount     NUMERIC(12,2) NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'new'
                     CHECK (status IN ('new','assembling','shipped','delivered','cancelled')),
    payment_status   TEXT NOT NULL DEFAULT 'unpaid' CHECK (payment_status IN ('unpaid','paid','failed')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_orders_shop_status ON orders(shop_id, status);

CREATE TABLE order_items (
    id           BIGSERIAL PRIMARY KEY,
    order_id     BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id   BIGINT NOT NULL REFERENCES products(id),
    product_name TEXT NOT NULL,                                       -- ДОРАБОТКА: название на момент покупки
    quantity     INTEGER NOT NULL CHECK (quantity > 0),
    unit_price   NUMERIC(12,2) NOT NULL                               -- цена на момент покупки
);
CREATE INDEX idx_items_order ON order_items(order_id);

CREATE TABLE order_status_history (
    id         BIGSERIAL PRIMARY KEY,
    order_id   BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    status     TEXT NOT NULL,
    changed_by BIGINT REFERENCES users(id),                           -- NULL = оформил покупатель
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_history_order ON order_status_history(order_id);

CREATE TABLE payments (
    id         BIGSERIAL PRIMARY KEY,
    order_id   BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    amount     NUMERIC(12,2) NOT NULL,
    status     TEXT NOT NULL CHECK (status IN ('pending','success','failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Служебные данные
CREATE TABLE moderation_records (
    id           BIGSERIAL PRIMARY KEY,
    shop_id      BIGINT REFERENCES shops(id) ON DELETE CASCADE,
    product_id   BIGINT REFERENCES products(id) ON DELETE CASCADE,
    moderator_id BIGINT NOT NULL REFERENCES users(id),
    decision     TEXT NOT NULL,
    reason       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (shop_id IS NOT NULL OR product_id IS NOT NULL)             -- ДОРАБОТКА: решение относится к чему-то одному из двух
);

CREATE TABLE ai_requests (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id  BIGINT REFERENCES products(id) ON DELETE SET NULL,
    type        TEXT NOT NULL CHECK (type IN ('description','price')),
    status      TEXT NOT NULL CHECK (status IN ('success','error')),
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Справочник категорий (стартовый набор)
INSERT INTO categories (name) VALUES ('Дом и сад'), ('Одежда'), ('Электроника'), ('Подарки');
INSERT INTO categories (name, parent_id) VALUES
    ('Садовые инструменты', (SELECT id FROM categories WHERE name = 'Дом и сад')),
    ('Семена и растения',   (SELECT id FROM categories WHERE name = 'Дом и сад')),
    ('Декор для дома',      (SELECT id FROM categories WHERE name = 'Дом и сад'));
