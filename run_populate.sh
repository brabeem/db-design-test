#!/bin/sh

echo "Building Go populate script..."
go mod download
go build -o populate populate.go

echo "Running populate script..."
./populate ./data/rubix.db

echo "Done! Database populated successfully."
echo "Cleaning up..."
rm ./populate

echo "Database is now available at: ./data/rubix.db"
