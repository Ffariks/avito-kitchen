CREATE TABLE restaurants (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    callback_url TEXT NOT NULL,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE menu_items (
    id            BIGSERIAL PRIMARY KEY,
    restaurant_id BIGINT NOT NULL REFERENCES restaurants (id),
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    price_cents   BIGINT NOT NULL CHECK (price_cents > 0),
    is_available  BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_menu_items_restaurant_id ON menu_items (restaurant_id);

CREATE TABLE orders (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL,
    restaurant_id BIGINT NOT NULL REFERENCES restaurants (id),
    status        TEXT NOT NULL DEFAULT 'created'
                  CHECK (status IN ('created', 'accepted', 'cooking', 'ready', 'delivered', 'rejected', 'cancelled')),
    total_cents   BIGINT NOT NULL CHECK (total_cents >= 0),
    comment       TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE INDEX idx_orders_restaurant_id ON orders (restaurant_id);
CREATE INDEX idx_orders_status ON orders (status);

CREATE TABLE order_items (
    id                    BIGSERIAL PRIMARY KEY,
    order_id              BIGINT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    menu_item_id          BIGINT NOT NULL REFERENCES menu_items (id),
    name                  TEXT NOT NULL,
    quantity              INT NOT NULL CHECK (quantity > 0),
    price_cents_at_order  BIGINT NOT NULL CHECK (price_cents_at_order > 0)
);
CREATE INDEX idx_order_items_order_id ON order_items (order_id);
