SELECT id, customer_id, status, is_paid, created_at FROM orders INNER JOIN order_items ON orders.id = order_items.order_id WHERE quantity > ?
