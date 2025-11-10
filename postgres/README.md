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
       NetworkCount      = 500   // 500 networks
       DevicesPerNetwork = 1000  // 1000 devices per network
       PointsPerDevice   = 500   // 500 points per device
   )
   // Total: ~250 million nodes
   ```

4. **Populate the database:**
   ```bash
   ./run_populate.sh
   ```
   
   **Note:** Populating 250 million rows will take several hours depending on your hardware.

## Database Structure

Same hierarchical structure as SQLite version:
- **Networks** → **Devices** → **Points**
- Each node has **ports** (input/output)
- **Edges** connect ports
- **Tags** provide metadata
- **Port values** store time-series data

## PostgreSQL Optimizations

### Configuration (No Resource Constraints)
- **CPU**: 8 cores allocated
- **Memory**: 16GB allocated
- **Shared Buffers**: 4GB
- **Effective Cache**: 12GB
- **Work Memory**: 10MB per operation
- **Parallel Workers**: 8 workers, 4 per query
- **WAL**: 1-4GB for write performance

### Advanced Indexes
- **B-tree indexes** for foreign keys and searches
- **BRIN indexes** for timestamp columns (time-series optimization)
- **Partial indexes** for unsynced port values
- **Composite indexes** for multi-column queries

## Benchmark Queries

Connect to the database:
```bash
docker exec -it rubix-postgres psql -U postgres -d rubix
```

### 1. Recursive Hierarchy Query
```sql
EXPLAIN (ANALYZE, BUFFERS)
WITH RECURSIVE nodes_hierarchy (id, name, type, description, level) AS (
    SELECT id, name, type, description, 0 
    FROM nodes 
    WHERE id = 'your-node-id'
    
    UNION ALL
    
    SELECT n.id, n.name, n.type, n.description, nh.level + 1
    FROM nodes_hierarchy nh
    JOIN nodes n ON n.parent_id = nh.id
    WHERE nh.level < 3
)
SELECT * FROM nodes_hierarchy;
```

### 2. Tag-Based Search
```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM nodes 
WHERE id IN (
    SELECT node_id FROM tags 
    WHERE tag_key = 'category' AND tag_value = 'medium'
);
```

### 3. Cascade Delete Performance
```sql
EXPLAIN (ANALYZE, BUFFERS)
DELETE FROM nodes WHERE id = 'your-node-id';
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


## Conclusion

PostgreSQL with proper configuration and indexing can efficiently handle hundreds of millions of rows in a central server environment. The combination of parallel query execution, advanced indexing strategies (BRIN, partial indexes), and high resource allocation makes it suitable for aggregating data from thousands of IoT edge devices running SQLite.

The key differences:
- **SQLite**: Resource-constrained, single-threaded, perfect for edge devices
- **PostgreSQL**: Highly parallel, feature-rich, ideal for central servers with complex queries and massive scale
