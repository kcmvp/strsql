SELECT * FROM orders WHERE EXISTS (SELECT 1 FROM order_items WHERE order_items.order_id = orders.id AND quantity > ?)
