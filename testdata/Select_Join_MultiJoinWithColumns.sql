SELECT o.id, p.name FROM orders AS o INNER JOIN order_items AS oi ON o.id = oi.order_id INNER JOIN products AS p ON oi.product_id = p.id WHERE p.price > ? ORDER BY o.created_at DESC LIMIT 10
