# SQLite Database Design for Edge Devices

This project demonstrates that SQLite can efficiently handle large amounts of hierarchical and graph data on edge devices when properly structured and indexed.

## Setup Instructions

1. **Start the SQLite container:**
   ```bash
   docker-compose up -d sqlite
   ```

2. **Configure data volume** (optional):
   
   Modify the counts in `populate.go` as needed:
   ```go
   const (
       NetworkCount      = 50
       DevicesPerNetwork = 50
       PointsPerDevice   = 100
   )
   ```

3. **Populate the database:**
   ```bash
   ./run_populate.sh
   ```

## Database Structure

The schema models a hierarchical graph structure:
- **Networks** → **Devices** → **Points**
- Each node has **ports** (input/output)
- **Edges** connect ports
- **Tags** provide metadata for searching
- **Port values** store time-series data

## Query Performance

### 1. Get all nodes under a parent up to a certain level

```sql
EXPLAIN QUERY PLAN
WITH RECURSIVE nodes_hierarchy (id, name, type, description, level) AS (
    SELECT id, name, type, description, 0 
    FROM nodes 
    WHERE id = "smvzghk49x8pxrp7gf5kmhmimk"
    
    UNION ALL
    
    SELECT nodes.id, nodes.name, nodes.type, nodes.description, nodes_hierarchy.level + 1
    FROM nodes_hierarchy 
    JOIN nodes ON nodes.parent_id = nodes_hierarchy.id
    WHERE level <= 3
)
SELECT * FROM nodes_hierarchy;
```

**Uses index:** `idx_nodes_parent_id`

### 2. Tag-based filtering

```sql
EXPLAIN QUERY PLAN 
SELECT * FROM nodes 
WHERE id IN (
    SELECT node_id FROM tags 
    WHERE tag_key = 'category' AND tag_value = 'medium'
);
```

**Uses index:** `idx_tags_tag_key_tag_value_node_id`

### 3. Cascade delete operation

```sql
EXPLAIN QUERY PLAN 
DELETE FROM nodes 
WHERE id = 'smvzghk49x8pxrp7gf5kmhmimk';
```

**Uses indexes:** Primary key on `nodes.id` and all foreign key indexes for cascade operations

## Important: Enable Foreign Keys

SQLite requires foreign keys to be enabled per connection:

```sql
PRAGMA foreign_keys = ON;
```

Run this before any operations when connecting via DBeaver or other clients.

## Conclusion

SQLite can be effectively used as a database on edge devices and can handle very large amounts of data if properly structured and indexed. It performs efficiently with graphical/hierarchical data when indexes are configured properly and the database is designed appropriately.
