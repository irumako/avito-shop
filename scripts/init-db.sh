#!/bin/bash
set -e

if [ -z "$DATABASE_URL" ]; then
  echo "Ошибка: переменная окружения DATABASE_URL не установлена."
  exit 1
fi

echo "Применение миграций..."
migrate -path ./migrations -database "$DATABASE_URL" up

sleep 2

echo "Создание пользователя admin и заполнение таблицы мерча..."
psql "$DATABASE_URL" <<'EOF'
-- Пароль "admin"
WITH admin_insert AS (
    INSERT INTO users (username, password_hash)
    VALUES ('admin', '$2y$10$lhwQ9O9.MIqv9ALICmgak.tUAF6SpNdeCxE1AcfDghbkrRwwvKXZC')
    ON CONFLICT (username) DO NOTHING
    RETURNING id
),
admin_id AS (
    SELECT id FROM admin_insert
    UNION
    SELECT id FROM users WHERE username = 'admin'
)

INSERT INTO wallets (user_id, balance)
SELECT id, 1000000
FROM admin_id
ON CONFLICT (user_id) DO NOTHING;

-- Заполняем таблицу мерча
INSERT INTO merch (name, cost) VALUES
  ('t-shirt', 80),
  ('cup', 20),
  ('book', 50),
  ('pen', 10),
  ('powerbank', 200),
  ('hoody', 300),
  ('umbrella', 200),
  ('socks', 10),
  ('wallet', 50),
  ('pink-hoody', 500)
ON CONFLICT (name) DO NOTHING;
EOF

echo "Инициализация базы данных завершена."
