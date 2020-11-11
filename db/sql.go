package db

const sqlCreateTableNodes = `
CREATE TABLE IF NOT EXISTS nodes (
	node_id INTEGER PRIMARY KEY,
	node_name VARCHAR(100) NOT NULL,
	node_alias VARCHAR(100) DEFAULT NULL,
	node_checked BOOLEAN DEFAULT FALSE,

	UNIQUE(node_alias)
)`

const sqlCreateTableEdges = `
CREATE TABLE IF NOT EXISTS edges (
	edge_id INTEGER PRIMARY KEY,
	origin_id INTEGER NOT NULL,
	dest_id INTEGER NOT NULL,

	FOREIGN KEY (origin_id)
		REFERENCES nodes (node_id)
		ON DELETE CASCADE

	FOREIGN KEY (dest_id)
		REFERENCES nodes (node_id)
		ON DELETE CASCADE

	CHECK(origin_id != dest_id)
	UNIQUE(origin_id, dest_id)
)`
