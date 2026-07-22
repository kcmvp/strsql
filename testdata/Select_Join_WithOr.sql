SELECT * FROM orders AS o INNER JOIN order_items AS oi ON o.id = oi.order_id WHERE (oi.product_id = ? OR oi.product_id = ?) LIMIT 1
