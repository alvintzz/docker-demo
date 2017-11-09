CREATE TABLE IF NOT EXISTS payments (
	id            serial,
	customer_name varchar(255) not null,
	primary key(product_id)
);

INSERT INTO payments(customer_name) VALUES ("alvin.tan@tokopedia.com");
INSERT INTO payments(customer_name) VALUES ("aliyyil.mustofa@tokopedia.com");
INSERT INTO payments(customer_name) VALUES ("yose.rizal@tokopedia.com");
INSERT INTO payments(customer_name) VALUES ("arief.rahmansyah@tokopedia.com");