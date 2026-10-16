-- Сначала заказы демо-заведения (order_items удалятся каскадом), иначе
-- удаление menu_items/restaurants упадёт на внешних ключах.
DELETE FROM orders WHERE restaurant_id = 1;
DELETE FROM menu_items WHERE restaurant_id = 1;
DELETE FROM restaurants WHERE id = 1;
