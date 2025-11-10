-- Test queries for PostgreSQL benchmark

-- IMPORTANT: These queries should use indexes efficiently

-- 1. Get all nodes under a parent up to a certain level (Recursive CTE)
EXPLAIN (ANALYZE, BUFFERS)
WITH RECURSIVE nodes_hierarchy (id, name, type, description, level) AS (
    SELECT id, name, type, description, 0 
    FROM nodes 
    WHERE id = 'REPLACE_WITH_ACTUAL_ID'
    
    UNION ALL
    
    SELECT n.id, n.name, n.type, n.description, nh.level + 1
    FROM nodes_hierarchy nh
    JOIN nodes n ON n.parent_id = nh.id
    WHERE nh.level < 3
)
SELECT * FROM nodes_hierarchy;

-- 2. Tag-based filtering
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM nodes 
WHERE id IN (
    SELECT node_id FROM tags 
    WHERE tag_key = 'category' AND tag_value = 'medium'
);

-- 3. Cascade delete operation (use EXPLAIN only, don't actually delete)
EXPLAIN (ANALYZE, BUFFERS)
DELETE FROM nodes 
WHERE id = 'REPLACE_WITH_ACTUAL_ID';

-- 4. Time-series query on port_values
EXPLAIN (ANALYZE, BUFFERS)
SELECT pv.*, n.name as node_name
FROM port_values pv
JOIN ports p ON pv.port_id = p.id
JOIN nodes n ON p.node_id = n.id
WHERE pv.timestamp > NOW() - INTERVAL '1 day'
  AND pv.is_synced = FALSE
ORDER BY pv.timestamp DESC
LIMIT 1000;

-- 5. Get unsynced port values count per node
EXPLAIN (ANALYZE, BUFFERS)
SELECT n.id, n.name, COUNT(pv.id) as unsynced_count
FROM nodes n
JOIN ports p ON p.node_id = n.id
JOIN port_values pv ON pv.port_id = p.id
WHERE pv.is_synced = FALSE
GROUP BY n.id, n.name
ORDER BY unsynced_count DESC
LIMIT 100;

-- Get some actual IDs to test with
SELECT id, name, type FROM nodes WHERE type = 'network' LIMIT 5;
SELECT id, name, type FROM nodes WHERE type = 'device' LIMIT 5;
SELECT id, name, type FROM nodes WHERE type = 'point' LIMIT 5;

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) AS indexes_size,
    n_live_tup AS row_count
FROM pg_stat_user_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
