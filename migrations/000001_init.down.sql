DROP TABLE IF EXISTS review_items;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS order_status_events;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS daily_order_counters;
DROP TABLE IF EXISTS delivery_slots;
DROP TABLE IF EXISTS menu_items;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS app_settings;

DROP FUNCTION IF EXISTS touch_updated_at;

DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_mode;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS user_role;
