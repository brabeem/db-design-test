-- PostgreSQL Schema for Central Server
-- Optimized for high-volume data collection from IoT devices

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- nodes, ports, port_values, edges, tags
-- operation 1: efficiently delete a node and all its child nodes, ports, port_values, edges
-- operation 2: efficiently get nodes under a parent node, level by level
-- operation 3: efficiently search tags

CREATE TABLE IF NOT EXISTS nodes(
    id VARCHAR(26) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    parent_id VARCHAR(26) REFERENCES nodes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ports(
    id VARCHAR(26) PRIMARY KEY,
    node_id VARCHAR(26) REFERENCES nodes(id) ON DELETE CASCADE,
    port_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS port_values(
    id VARCHAR(26) PRIMARY KEY,
    port_id VARCHAR(26) REFERENCES ports(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    value_numeric DOUBLE PRECISION,
    value_text TEXT,
    value_boolean BOOLEAN,
    value_json JSONB,
    is_synced BOOLEAN DEFAULT FALSE,
    last_synced TIMESTAMP
);

CREATE TABLE IF NOT EXISTS edges(
    id VARCHAR(26) PRIMARY KEY,
    from_port_id VARCHAR(26) REFERENCES ports(id) ON DELETE CASCADE,
    to_port_id VARCHAR(26) REFERENCES ports(id) ON DELETE CASCADE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tags(
    node_id VARCHAR(26) REFERENCES nodes(id) ON DELETE CASCADE,
    tag_key VARCHAR(100) NOT NULL,
    tag_value VARCHAR(255) NOT NULL,
    PRIMARY KEY (node_id, tag_key, tag_value)
);

-- Indexes for hierarchical queries
CREATE INDEX IF NOT EXISTS idx_nodes_parent_id ON nodes(parent_id);

-- Indexes for cascade delete operations and joins
CREATE INDEX IF NOT EXISTS idx_ports_node_id ON ports(node_id);
CREATE INDEX IF NOT EXISTS idx_port_values_port_id ON port_values(port_id);
CREATE INDEX IF NOT EXISTS idx_edges_from_port_id ON edges(from_port_id);
CREATE INDEX IF NOT EXISTS idx_edges_to_port_id ON edges(to_port_id);
CREATE INDEX IF NOT EXISTS idx_tags_node_id ON tags(node_id);

-- Indexes for tag search operations
CREATE INDEX IF NOT EXISTS idx_tags_tag_key_tag_value_node_id ON tags(tag_key, tag_value, node_id);