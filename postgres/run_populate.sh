#!/bin/sh

echo "Building Go populate script..."
cd "$(dirname "$0")"
go mod download
go build -o populate populate.go

echo "Running populate script..."
echo "This will create ~250 million nodes. It may take several hours."
echo "Press Ctrl+C to cancel or Enter to continue..."
read

./populate

echo "Done! Database populated successfully."
echo "Cleaning up..."
rm ./populate

echo "Database statistics:"
docker exec rubix-postgres psql -U postgres -d rubix -c "
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size,
    n_live_tup AS row_count
FROM pg_stat_user_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
"
