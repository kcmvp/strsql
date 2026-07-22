SELECT * FROM orders AS o RIGHT JOIN order_items AS oi ON o.id = oi.order_id WHERE oi.product_id = ? LIMIT 1
