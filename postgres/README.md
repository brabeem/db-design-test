# PostgreSQL Database Design for Central Server

This setup demonstrates PostgreSQL's capability to handle massive amounts of hierarchical and graph data from IoT devices in a central server environment with no resource constraints.

## Setup Instructions

1. **Start the PostgreSQL container:**
   ```bash
   cd postgres
   docker-compose up -d
   ```

2. **Wait for PostgreSQL to be ready:**
   ```bash
   docker logs -f rubix-postgres
   # Wait for "database system is ready to accept connections"
   ```

3. **Configure data volume** (optional):
   
   Modify the counts in `populate.go`:
   ```go
   const (
       NetworkCount      = 100   // 100 networks
       DevicesPerNetwork = 100   // 100 devices per network
       PointsPerDevice   = 1000  // 1000 points per device
   )
   // Total: ~10 million nodes
   ```

4. **Populate the database:**
   ```bash
   ./run_populate.sh
   ```
   
   **Note:** Populating 10 million rows will take 2-4 hours depending on your hardware.

## Database Structure

Same hierarchical structure as SQLite version:
- **Networks** → **Devices** → **Points**
- Each node has **ports** (input/output)
- **Edges** connect ports
- **Tags** provide metadata
- **Port values** store time-series data


## Benchmark Queries

Connect to the database:
```bash
docker exec -it rubix-postgres psql -U postgres -d rubix
```

### 1. Recursive Hierarchy Query

**Standard recursive CTE approach:**
```sql
EXPLAIN (ANALYZE, BUFFERS)
WITH RECURSIVE nodes_hierarchy (id, name, type, description, level) AS (
    SELECT id, name, type, description, 0 
    FROM nodes 
    WHERE id = 'amamz0ej5vgt4ib0moc1qaqk3m'
    
    UNION ALL
    
    SELECT n.id, n.name, n.type, n.description, nh.level + 1
    FROM nodes_hierarchy nh
    JOIN nodes n ON n.parent_id = nh.id
    WHERE nh.level < 3
)
SELECT * FROM nodes_hierarchy;
```

**Optimized approach using multiple CTEs (significantly faster):**

This approach is expected to use bitmap heap scans instead of index scans in case of large number of index scans, avoiding repeated index-to-heap lookups which are inefficient for large result sets.

```sql
EXPLAIN (ANALYZE, BUFFERS)
WITH base AS (
    SELECT id, name, type, description, 0 AS level
    FROM nodes
    WHERE id = 'amamz0ej5vgt4ib0moc1qaqk3m'
),
level_1 AS (
    SELECT n.id, n.name, n.type, n.description, 1 AS level
    FROM nodes n
    WHERE parent_id IN (SELECT id FROM base)
),
level_2 AS (
    SELECT n.id, n.name, n.type, n.description, 2 AS level
    FROM nodes n
    WHERE parent_id IN (SELECT id FROM level_1)
),
level_3 AS (
    SELECT n.id, n.name, n.type, n.description, 3 AS level
    FROM nodes n
    WHERE parent_id IN (SELECT id FROM level_2)
)
SELECT * FROM base
UNION ALL SELECT * FROM level_1
UNION ALL SELECT * FROM level_2
UNION ALL SELECT * FROM level_3;
```

**Performance note:** The multi-CTE approach performs significantly better on large datasets as it leverages bitmap heap scans, which are more efficient when fetching multiple rows.


### 2. Tag-Based Search

**Single tag search:**
```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM nodes 
WHERE id IN (
    SELECT node_id FROM tags 
    WHERE tag_key = 'category' AND tag_value = 'medium'
);
```

**Multiple tag search (AND condition - nodes must have ALL specified tags):**

This query efficiently finds nodes that satisfy multiple tag criteria, ensuring all specified tag key-value pairs are present.
```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT node_id
FROM tags
WHERE 
      (tag_key = 'gauge_37'  AND tag_value = 'stopped_517')
   OR (tag_key = 'info_83'   AND tag_value = 'hw-rev-a_489')
   OR (tag_key = 'range_44'  AND tag_value = 'secondary_89')
   OR (tag_key = 'trace_34'  AND tag_value = 'rack-1_253')
GROUP BY node_id
HAVING COUNT(DISTINCT tag_key) = 4;
```

If the query consist of key only rather than key and value , then , it return millions of rows so , shouldn't be invovled in the
first ORing , and should be used in the latter result set only.

```sql
EXPLAIN (ANALYZE, BUFFERS)
WITH filtered_nodes AS (
  SELECT node_id
  FROM tags
  WHERE 
        (tag_key = 'gauge_37'  AND tag_value = 'stopped_517')
     OR (tag_key = 'info_83'   AND tag_value = 'hw-rev-a_489')
     OR (tag_key = 'range_44'  AND tag_value = 'secondary_89')
  GROUP BY node_id
  HAVING COUNT(DISTINCT (tag_key, tag_value)) = 3
)
SELECT nd.*, tg.*
FROM nodes nd
JOIN filtered_nodes fn ON nd.id = fn.node_id
JOIN tags tg ON nd.id = tg.node_id
WHERE tg.tag_key = 'trace_34';
```

**Performance note:** This approach performs significantly better than multiple JOINs or subqueries, as it uses a single index scan with grouping to filter results.

### 3. Cascade Delete Performance
```sql
EXPLAIN (ANALYZE, BUFFERS)
DELETE FROM nodes WHERE id = 'tfm21tzp8pbo6q09v5i4r9g3no';
```

The delete is really heavy operation if done on top level nodes, which can be realised from the plan below; This is currently very slow , looking for changes in  schema 
to make this faster.
```      
------------------------------------------------------------------------------------------------------------------------
 Delete on nodes  (cost=0.56..2.78 rows=0 width=0) (actual time=2.517..2.517 rows=0 loops=1)
   Buffers: shared hit=10 read=5 dirtied=2
   ->  Index Scan using nodes_pkey on nodes  (cost=0.56..2.78 rows=1 width=6) (actual time=0.689..0.692 rows=1 loops=1)
         Index Cond: ((id)::text = 'tfm21tzp8pbo6q09v5i4r9g3no'::text)
         Buffers: shared hit=3 read=2
 Planning Time: 0.085 ms
 Trigger for constraint nodes_parent_id_fkey on nodes: time=9458.431 calls=100101
 Trigger for constraint ports_node_id_fkey on nodes: time=35877.566 calls=100101
 Trigger for constraint tags_node_id_fkey on nodes: time=43446.110 calls=100101
 Trigger for constraint port_values_port_id_fkey on ports: time=58943.024 calls=200202
 Trigger for constraint edges_from_port_id_fkey on ports: time=9651.115 calls=200202
 Trigger for constraint edges_to_port_id_fkey on ports: time=8228.618 calls=200202
 Execution Time: 165867.388 ms
(13 rows)
```

## Monitoring

### Database Size
```bash
docker exec rubix-postgres psql -U postgres -d rubix -c "
SELECT 
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size,
    n_live_tup AS row_count
FROM pg_stat_user_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
"
```

### Index Usage Statistics
```sql
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as scans,
    pg_size_pretty(pg_relation_size(indexrelid)) as size
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```


## Performance Insights

The optimized queries demonstrate that PostgreSQL performs exceptionally well with proper query patterns:

- **Hierarchical queries**: Multi-CTE approach with bitmap heap scans significantly outperforms recursive CTEs
- **Tag searches**: GROUP BY with HAVING performs much better than multiple JOINs for multi-tag filtering
- **Index strategy**: Composite indexes combined with bitmap scans provide optimal performance for large result sets

## Conclusion

PostgreSQL with proper configuration and indexing can efficiently handle tens of millions of rows in a central server environment. The combination of parallel query execution, advanced indexing strategies (BRIN, partial indexes), and high resource allocation makes it suitable for aggregating data from thousands of IoT edge devices running SQLite.

**Key differences:**
- **SQLite**: Resource-constrained, single-threaded, perfect for edge devices
- **PostgreSQL**: Highly parallel, feature-rich, ideal for central servers with complex queries and massive scale

**Query optimization matters:** Proper query patterns (bitmap scans vs index scans, CTEs vs recursive CTEs) can yield 10x-100x performance improvements on large datasets.
