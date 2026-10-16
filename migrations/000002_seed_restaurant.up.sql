-- Закрытый список заведений (см. ТЗ, раздел "Заведения"): в MVP заведения
-- не регистрируются сами, площадка добавляет их вручную/миграцией.
INSERT INTO restaurants (name, description, callback_url, is_active)
VALUES ('Тестовая кухня', 'Демонстрационное заведение для интеграции', 'http://restaurant-service:9090', true);

INSERT INTO menu_items (restaurant_id, name, description, price_cents, is_available)
VALUES
    (1, 'Борщ', 'Классический борщ со сметаной', 35000, true),
    (1, 'Плов', 'Плов с бараниной', 42000, true),
    (1, 'Морс', 'Клюквенный морс 0.5л', 12000, false);
