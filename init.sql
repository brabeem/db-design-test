-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- Enable recursive triggers (CRITICAL for cascading soft deletes)
PRAGMA recursive_triggers = ON;

--  nodes, ports , port_values, edges , tags  
--  operation 1 : should be able to effieciently delete a node and all its child nodes, ports, port_values, edges
--  operation 2 : should be able to efficiently get nodes under a parent node,level by level given input level(recursively created index for that)
--  operation 3 : efficiently search tags

CREATE TABLE IF NOT EXISTS nodes(
    id VARCHAR(26) PRIMARY KEY,
    deleted_at TIMESTAMP DEFAULT NULL,                  
    type VARCHAR(50) NOT NULL,  -- e.g : 'network', 'devices', 'points'
    parent_id VARCHAR(26) REFERENCES nodes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT
);     

CREATE TABLE IF NOT EXISTS ports(
    id VARCHAR(26) PRIMARY KEY,
    deleted_at TIMESTAMP DEFAULT NULL,
    node_id VARCHAR(26) REFERENCES nodes(id) ON DELETE CASCADE,
    port_type VARCHAR(50) NOT NULL, -- e.g : 'input', 'output'
    name VARCHAR(255) NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS port_values(
    id VARCHAR(26) PRIMARY KEY,
    deleted_at TIMESTAMP DEFAULT NULL,
    port_id VARCHAR(26) REFERENCES ports(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    value_numeric DOUBLE PRECISION,
    value_text TEXT,
    value_boolean BOOLEAN,
    value_json TEXT,
    is_synced BOOLEAN DEFAULT FALSE,
    last_synced TIMESTAMP
);

CREATE TABLE IF NOT EXISTS edges(
    id VARCHAR(26) PRIMARY KEY,
    deleted_at TIMESTAMP DEFAULT NULL,
    from_port_id VARCHAR(26) REFERENCES ports(id) ON DELETE CASCADE,
    to_port_id VARCHAR(26) REFERENCES ports(id) ON DELETE CASCADE,
    description TEXT
);

CREATE TABLE IF NOT EXISTS tags(
    node_id VARCHAR(26) REFERENCES nodes(id) ON DELETE CASCADE,
    deleted_at TIMESTAMP DEFAULT NULL,
    tag_key VARCHAR(100) NOT NULL,
    tag_value VARCHAR(255) NOT NULL DEFAULT '',
    PRIMARY KEY (node_id, tag_key, tag_value)
);

-- These are better than partial indexes for multi-column WHERE clauses
CREATE INDEX IF NOT EXISTS idx_nodes_parent_id_deleted ON nodes(parent_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_ports_node_id_deleted ON ports(node_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_port_values_port_id_deleted ON port_values(port_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_edges_from_port_deleted ON edges(from_port_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_edges_to_port_deleted ON edges(to_port_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_tags_node_id_deleted ON tags(node_id, deleted_at);

-- Tag search indexes (composite for better performance)
CREATE INDEX IF NOT EXISTS idx_tags_key_value_node_deleted ON tags(tag_key, tag_value, deleted_at, node_id );

-- Triggers to set deleted_at timestamp on soft delete
CREATE TRIGGER IF NOT EXISTS cascade_soft_delete_nodes_on_nodes_delete
AFTER UPDATE OF deleted_at ON nodes 
FOR EACH ROW 
WHEN NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL
BEGIN
    UPDATE nodes SET deleted_at = NEW.deleted_at 
    WHERE parent_id = NEW.id AND deleted_at IS NULL;
END;

CREATE TRIGGER IF NOT EXISTS cascade_soft_delete_ports_on_nodes_delete
AFTER UPDATE OF deleted_at ON nodes 
FOR EACH ROW 
WHEN NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL
BEGIN
    UPDATE ports SET deleted_at = NEW.deleted_at 
    WHERE node_id = NEW.id AND deleted_at IS NULL;
END;

CREATE TRIGGER IF NOT EXISTS cascade_soft_delete_port_values_on_ports_delete
AFTER UPDATE OF deleted_at ON ports 
FOR EACH ROW 
WHEN NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL
BEGIN
    UPDATE port_values SET deleted_at = NEW.deleted_at 
    WHERE port_id = NEW.id AND deleted_at IS NULL;
END;

CREATE TRIGGER IF NOT EXISTS cascade_soft_delete_edges_on_ports_delete
AFTER UPDATE OF deleted_at ON ports 
FOR EACH ROW 
WHEN NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL
BEGIN
    UPDATE edges SET deleted_at = NEW.deleted_at 
    WHERE (from_port_id = NEW.id OR to_port_id = NEW.id) AND deleted_at IS NULL;
END;

CREATE TRIGGER IF NOT EXISTS cascade_soft_delete_tags_on_nodes_delete
AFTER UPDATE OF deleted_at ON nodes 
FOR EACH ROW 
WHEN NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL
BEGIN
    UPDATE tags SET deleted_at = NEW.deleted_at 
    WHERE node_id = NEW.id AND deleted_at IS NULL;
END;


-- Trigger to restore deleted_at to NULL on parent node update
CREATE TRIGGER IF NOT EXISTS cascade_restore_nodes_on_nodes_restore
AFTER UPDATE OF deleted_at ON nodes 
FOR EACH ROW 
WHEN NEW.deleted_at IS NULL AND OLD.deleted_at IS NOT NULL
BEGIN
    UPDATE nodes SET deleted_at = NULL 
    WHERE parent_id = NEW.id AND deleted_at IS NOT NULL;
END;    

CREATE TRIGGER IF NOT EXISTS cascade_restore_ports_on_nodes_restore
AFTER UPDATE OF deleted_at ON nodes 
FOR EACH ROW 
WHEN NEW.deleted_at IS NULL AND OLD.deleted_at IS NOT NULL
BEGIN
    UPDATE ports SET deleted_at = NULL 
    WHERE node_id = NEW.id AND deleted_at IS NOT NULL;
END;

CREATE TRIGGER IF NOT EXISTS cascade_restore_port_values_on_ports_restore
AFTER UPDATE OF deleted_at ON ports 
FOR EACH ROW 
WHEN NEW.deleted_at IS NULL AND OLD.deleted_at IS NOT NULL
BEGIN
    UPDATE port_values SET deleted_at = NULL 
    WHERE port_id = NEW.id AND deleted_at IS NOT NULL;
END;

CREATE TRIGGER IF NOT EXISTS cascade_restore_edges_on_ports_restore
AFTER UPDATE OF deleted_at ON ports 
FOR EACH ROW 
WHEN NEW.deleted_at IS NULL AND OLD.deleted_at IS NOT NULL
BEGIN
    UPDATE edges SET deleted_at = NULL 
    WHERE (from_port_id = NEW.id OR to_port_id = NEW.id) AND deleted_at IS NOT NULL;
END;

CREATE TRIGGER IF NOT EXISTS cascade_restore_tags_on_nodes_restore
AFTER UPDATE OF deleted_at ON nodes 
FOR EACH ROW 
WHEN NEW.deleted_at IS NULL AND OLD.deleted_at IS NOT NULL
BEGIN
    UPDATE tags SET deleted_at = NULL 
    WHERE node_id = NEW.id AND deleted_at IS NOT NULL;
END;