-- ============ EXTENSIONS ============
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

-- ============ ENUMS ============
CREATE TYPE user_role AS ENUM ('CUSTOMER','STAFF','ADMIN');
CREATE TYPE order_status AS ENUM (
  'PLACED','ACCEPTED','PREPARING','READY',
  'OUT_FOR_DELIVERY','DELIVERED',
  'CANCELLED_BY_USER','CANCELLED_BY_ADMIN'
);
CREATE TYPE payment_mode   AS ENUM ('COD','ONLINE');
CREATE TYPE payment_status AS ENUM ('PENDING','PAID','FAILED','REFUNDED');

-- ============ USERS ============
CREATE TABLE users (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  name           TEXT        NOT NULL CHECK (length(trim(name)) BETWEEN 2 AND 60),
  email          CITEXT      NOT NULL UNIQUE,
  phone          TEXT        NOT NULL UNIQUE CHECK (phone ~ '^[6-9][0-9]{9}$'),
  password_hash  TEXT        NOT NULL,
  role           user_role   NOT NULL DEFAULT 'CUSTOMER',
  is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_role ON users(role) WHERE is_active;

-- ============ REFRESH TOKENS ============
CREATE TABLE refresh_tokens (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  TEXT        NOT NULL UNIQUE,
  family_id   UUID        NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  revoked_at  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_rt_user   ON refresh_tokens(user_id);
CREATE INDEX idx_rt_family ON refresh_tokens(family_id);

-- ============ LOCATIONS ============
CREATE TABLE locations (
  id                 UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  code               TEXT    NOT NULL UNIQUE,
  name               TEXT    NOT NULL,
  delivery_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
  delivery_fee_paise INT     NOT NULL DEFAULT 0 CHECK (delivery_fee_paise >= 0),
  sort_order         INT     NOT NULL DEFAULT 0,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ MENU ============
CREATE TABLE categories (
  id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT    NOT NULL UNIQUE,
  sort_order INT     NOT NULL DEFAULT 0,
  is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE menu_items (
  id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  category_id   UUID    NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
  name          TEXT    NOT NULL,
  description   TEXT    NOT NULL DEFAULT '',
  price_paise   INT     NOT NULL CHECK (price_paise > 0),
  image_url     TEXT,
  is_veg        BOOLEAN NOT NULL DEFAULT TRUE,
  is_available  BOOLEAN NOT NULL DEFAULT TRUE,
  is_active     BOOLEAN NOT NULL DEFAULT TRUE,
  rating_avg    NUMERIC(2,1) NOT NULL DEFAULT 0,
  rating_count  INT     NOT NULL DEFAULT 0,
  sort_order    INT     NOT NULL DEFAULT 0,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_items_cat ON menu_items(category_id) WHERE is_active;

-- ============ SLOTS ============
CREATE TABLE delivery_slots (
  id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id     UUID    NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  slot_date       DATE    NOT NULL,
  start_time      TIME    NOT NULL,
  end_time        TIME    NOT NULL,
  capacity        INT     NOT NULL CHECK (capacity > 0),
  booked_count    INT     NOT NULL DEFAULT 0 CHECK (booked_count >= 0),
  cutoff_minutes  INT     NOT NULL DEFAULT 30 CHECK (cutoff_minutes >= 0),
  is_active       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uq_slot UNIQUE (location_id, slot_date, start_time),
  CONSTRAINT ck_time CHECK (end_time > start_time),
  CONSTRAINT ck_capacity CHECK (booked_count <= capacity)
);
CREATE INDEX idx_slots_lookup ON delivery_slots(location_id, slot_date) WHERE is_active;

-- ============ SHORT CODE COUNTER ============
CREATE TABLE daily_order_counters (
  counter_date DATE    NOT NULL,
  sequence     INT     NOT NULL DEFAULT 0,
  PRIMARY KEY  (counter_date)
);

-- ============ ORDERS ============
CREATE TABLE orders (
  id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  short_code          TEXT          NOT NULL UNIQUE,
  user_id             UUID          NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  location_id         UUID          NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
  slot_id             UUID          NOT NULL REFERENCES delivery_slots(id) ON DELETE RESTRICT,
  status              order_status  NOT NULL DEFAULT 'PLACED',
  payment_mode        payment_mode  NOT NULL DEFAULT 'COD',
  payment_status      payment_status NOT NULL DEFAULT 'PENDING',
  subtotal_paise      INT           NOT NULL CHECK (subtotal_paise >= 0),
  delivery_fee_paise  INT           NOT NULL DEFAULT 0,
  total_paise         INT           NOT NULL CHECK (total_paise >= 0),
  notes               TEXT          NOT NULL DEFAULT '' CHECK (length(notes) <= 200),
  cancel_deadline_at  TIMESTAMPTZ   NOT NULL,
  placed_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
  delivered_at        TIMESTAMPTZ,
  cancelled_at        TIMESTAMPTZ,
  cancel_reason       TEXT,
  updated_at          TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX idx_orders_user  ON orders(user_id, placed_at DESC);
CREATE INDEX idx_orders_slot  ON orders(slot_id);
CREATE INDEX idx_orders_queue ON orders(status, placed_at)
  WHERE status IN ('PLACED','ACCEPTED','PREPARING','READY','OUT_FOR_DELIVERY');

CREATE TABLE order_items (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id             UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  menu_item_id         UUID NOT NULL REFERENCES menu_items(id) ON DELETE RESTRICT,
  name_snapshot        TEXT NOT NULL,
  price_snapshot_paise INT  NOT NULL CHECK (price_snapshot_paise > 0),
  quantity             INT  NOT NULL CHECK (quantity BETWEEN 1 AND 20),
  line_total_paise     INT  NOT NULL
);
CREATE INDEX idx_oi_order ON order_items(order_id);

CREATE TABLE order_status_events (
  id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id    UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  from_status order_status,
  to_status   order_status  NOT NULL,
  actor_id    UUID          REFERENCES users(id),
  note        TEXT          NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX idx_ose_order ON order_status_events(order_id, created_at);

-- ============ REVIEWS ============
CREATE TABLE reviews (
  id             UUID     PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id       UUID     NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
  user_id        UUID     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rating         SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment        TEXT     NOT NULL DEFAULT '' CHECK (length(comment) <= 500),
  is_hidden      BOOLEAN  NOT NULL DEFAULT FALSE,
  editable_until TIMESTAMPTZ NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_reviews_public ON reviews(created_at DESC) WHERE NOT is_hidden;

CREATE TABLE review_items (
  review_id    UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
  menu_item_id UUID NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
  PRIMARY KEY  (review_id, menu_item_id)
);

-- ============ SETTINGS ============
CREATE TABLE app_settings (
  key        TEXT PRIMARY KEY,
  value      TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO app_settings(key, value) VALUES
  ('delivery_enabled', 'true'),
  ('kitchen_open',     'true'),
  ('announcement',     '');

-- ============ TRIGGERS ============
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg_users_touch  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER tg_items_touch  BEFORE UPDATE ON menu_items
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER tg_orders_touch BEFORE UPDATE ON orders
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER tg_reviews_touch BEFORE UPDATE ON reviews
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- ============ SEED LOCATIONS ============
INSERT INTO locations(code, name, sort_order) VALUES
  ('LJ',     'LJ Campus',  1),
  ('TTECH',  'TTECH Park', 2),
  ('STRATA', 'Strata',     3);

INSERT INTO categories(name, sort_order) VALUES
  ('Veg Rolls',     1),
  ('Non-Veg Rolls', 2),
  ('Sides',         3),
  ('Beverages',     4);
