package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

const (
	// Target: ~10 million nodes (optimized for Mac with 512GB disk)
	// Structure: Networks -> Devices -> Points
	// Expected DB size: ~80-100GB, Time: 2-4 hours
	NetworkCount      = 100  // 100 networks
	DevicesPerNetwork = 100  // 100 devices per network = 10k devices
	PointsPerDevice   = 1000 // 1000 points per device = 10M points
	PortsPerNode      = 2    // input and output ports per node
	BatchSize         = 5000 // Batch inserts for performance
	NumWorkers        = 8    // Parallel workers for data generation
)

var (
	// Generate large tag vocabulary for realistic IoT scenarios
	tagKeys   []string
	tagValues []string
)

func init() {
	// Generate 100 tag keys
	tagKeyPrefixes := []string{"location", "category", "priority", "status", "environment", "region", "zone", "criticality",
		"vendor", "model", "protocol", "firmware", "hardware", "software", "network", "building", "floor", "room", "rack",
		"cabinet", "sensor", "actuator", "controller", "gateway", "interface", "port", "channel", "mode", "state", "alarm",
		"warning", "error", "info", "debug", "trace", "metric", "counter", "gauge", "histogram", "measurement", "reading",
		"setpoint", "threshold", "limit", "range", "scale", "unit", "dimension", "attribute", "property", "feature"}

	for i := 0; i < 100; i++ {
		prefix := tagKeyPrefixes[i%len(tagKeyPrefixes)]
		tagKeys = append(tagKeys, fmt.Sprintf("%s_%d", prefix, i))
	}

	// Generate 1000 tag values
	tagValuePrefixes := []string{"production", "staging", "development", "testing", "qa", "demo", "backup", "archive",
		"high", "medium", "low", "critical", "warning", "info", "normal", "active", "inactive", "enabled", "disabled",
		"online", "offline", "running", "stopped", "paused", "pending", "failed", "success", "error", "timeout",
		"north", "south", "east", "west", "northeast", "northwest", "southeast", "southwest", "central", "edge",
		"zone-a", "zone-b", "zone-c", "zone-d", "region-1", "region-2", "region-3", "region-4", "region-5",
		"floor-1", "floor-2", "floor-3", "building-a", "building-b", "site-1", "site-2", "rack-1", "rack-2",
		"modbus", "bacnet", "mqtt", "opcua", "http", "coap", "lorawan", "zigbee", "ble", "wifi", "ethernet",
		"temp", "humidity", "pressure", "flow", "level", "voltage", "current", "power", "energy", "frequency",
		"open", "closed", "on", "off", "auto", "manual", "local", "remote", "master", "slave", "primary", "secondary",
		"v1.0", "v2.0", "v3.0", "hw-rev-a", "hw-rev-b", "fw-latest", "config-default", "mode-standard", "mode-advanced"}

	for i := 0; i < 1000; i++ {
		prefix := tagValuePrefixes[i%len(tagValuePrefixes)]
		tagValues = append(tagValues, fmt.Sprintf("%s_%d", prefix, i))
	}
}

func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 26)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Database connection
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5433"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=postgres password=postgres dbname=rubix sslmode=disable", dbHost, dbPort)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Configure connection pool for high performance
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	log.Println("Connected to PostgreSQL")
	log.Println("Starting data population...")
	log.Printf("Target: %d networks, %d devices, ~%d million points",
		NetworkCount, NetworkCount*DevicesPerNetwork, (NetworkCount*DevicesPerNetwork*PointsPerDevice)/1000000)

	startTime := time.Now()

	// Populate in phases
	log.Println("\n=== Phase 1: Creating Networks ===")
	networks := createNetworks(db)

	log.Println("\n=== Phase 2: Creating Devices ===")
	devices := createDevices(db, networks)

	log.Println("\n=== Phase 3: Creating Points (This will take a while...) ===")
	createPoints(db, devices)

	log.Println("\n=== Phase 4: Creating Edges ===")
	createEdges(db)

	log.Println("\n=== Phase 5: Analyzing Tables ===")
	analyzeTables(db)

	duration := time.Since(startTime)

	// Get final counts
	var nodeCount, portCount, portValueCount, tagCount, edgeCount int64
	db.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&nodeCount)
	db.QueryRow("SELECT COUNT(*) FROM ports").Scan(&portCount)
	db.QueryRow("SELECT COUNT(*) FROM port_values").Scan(&portValueCount)
	db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
	db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&edgeCount)

	log.Println("\n=== Data Population Complete ===")
	log.Printf("Total Nodes: %d", nodeCount)
	log.Printf("Total Ports: %d", portCount)
	log.Printf("Total Port Values: %d", portValueCount)
	log.Printf("Total Tags: %d", tagCount)
	log.Printf("Total Edges: %d", edgeCount)
	log.Printf("Total Time: %s", duration)
	log.Printf("Average Rate: %.0f nodes/second", float64(nodeCount)/duration.Seconds())
}

func createNetworks(db *sql.DB) []string {
	networks := make([]string, 0, NetworkCount)

	tx, _ := db.Begin()
	defer tx.Rollback()

	nodeStmt, _ := tx.Prepare("INSERT INTO nodes (id, type, parent_id, name, description) VALUES ($1, $2, $3, $4, $5)")
	defer nodeStmt.Close()
	portStmt, _ := tx.Prepare("INSERT INTO ports (id, node_id, port_type, name, description) VALUES ($1, $2, $3, $4, $5)")
	defer portStmt.Close()
	tagStmt, _ := tx.Prepare("INSERT INTO tags (node_id, tag_key, tag_value) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING")
	defer tagStmt.Close()

	for i := 1; i <= NetworkCount; i++ {
		networkID := generateID()
		networks = append(networks, networkID)

		nodeStmt.Exec(networkID, "network", nil, fmt.Sprintf("Network-%d", i), fmt.Sprintf("Central network %d", i))

		// Create ports
		inputPortID := generateID()
		outputPortID := generateID()
		portStmt.Exec(inputPortID, networkID, "input", fmt.Sprintf("Input-%s", networkID[:8]), "Input port")
		portStmt.Exec(outputPortID, networkID, "output", fmt.Sprintf("Output-%s", networkID[:8]), "Output port")

		// Create tags (5-15 tags per network for realistic metadata)
		numTags := rand.Intn(11) + 5
		for t := 0; t < numTags; t++ {
			tagStmt.Exec(networkID, tagKeys[rand.Intn(len(tagKeys))], tagValues[rand.Intn(len(tagValues))])
		}

		if i%100 == 0 {
			log.Printf("Created %d/%d networks", i, NetworkCount)
		}
	}

	tx.Commit()
	log.Printf("Created %d networks", NetworkCount)
	return networks
}

func createDevices(db *sql.DB, networks []string) []string {
	devices := make([]string, 0, NetworkCount*DevicesPerNetwork)

	for netIdx, networkID := range networks {
		tx, _ := db.Begin()

		nodeStmt, _ := tx.Prepare("INSERT INTO nodes (id, type, parent_id, name, description) VALUES ($1, $2, $3, $4, $5)")
		portStmt, _ := tx.Prepare("INSERT INTO ports (id, node_id, port_type, name, description) VALUES ($1, $2, $3, $4, $5)")
		tagStmt, _ := tx.Prepare("INSERT INTO tags (node_id, tag_key, tag_value) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING")

		for d := 1; d <= DevicesPerNetwork; d++ {
			deviceID := generateID()
			devices = append(devices, deviceID)

			nodeStmt.Exec(deviceID, "device", networkID, fmt.Sprintf("Device-%d-%d", netIdx+1, d), fmt.Sprintf("Device %d", d))

			// Create ports
			inputPortID := generateID()
			outputPortID := generateID()
			portStmt.Exec(inputPortID, deviceID, "input", fmt.Sprintf("Input-%s", deviceID[:8]), "Input port")
			portStmt.Exec(outputPortID, deviceID, "output", fmt.Sprintf("Output-%s", deviceID[:8]), "Output port")

			// Create tags (8-20 tags per device)
			numTags := rand.Intn(13) + 8
			for t := 0; t < numTags; t++ {
				tagStmt.Exec(deviceID, tagKeys[rand.Intn(len(tagKeys))], tagValues[rand.Intn(len(tagValues))])
			}
		}

		nodeStmt.Close()
		portStmt.Close()
		tagStmt.Close()
		tx.Commit()

		log.Printf("Created devices for network %d/%d (%d total devices)", netIdx+1, NetworkCount, len(devices))
	}

	log.Printf("Created %d devices", len(devices))
	return devices
}

func createPoints(db *sql.DB, devices []string) {
	var wg sync.WaitGroup
	deviceChan := make(chan string, NumWorkers*2)

	// Start workers
	for w := 0; w < NumWorkers; w++ {
		wg.Add(1)
		go pointWorker(db, deviceChan, &wg, w)
	}

	// Feed devices to workers
	for _, deviceID := range devices {
		deviceChan <- deviceID
	}
	close(deviceChan)

	wg.Wait()
	log.Printf("Created points for all %d devices", len(devices))
}

func pointWorker(db *sql.DB, deviceChan <-chan string, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	processedCount := 0
	for deviceID := range deviceChan {
		tx, _ := db.Begin()

		nodeStmt, _ := tx.Prepare("INSERT INTO nodes (id, type, parent_id, name, description) VALUES ($1, $2, $3, $4, $5)")
		portStmt, _ := tx.Prepare("INSERT INTO ports (id, node_id, port_type, name, description) VALUES ($1, $2, $3, $4, $5)")
		portValueStmt, _ := tx.Prepare("INSERT INTO port_values (id, port_id, timestamp, value_numeric, value_text, value_boolean, is_synced) VALUES ($1, $2, $3, $4, $5, $6, $7)")
		tagStmt, _ := tx.Prepare("INSERT INTO tags (node_id, tag_key, tag_value) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING")

		for p := 1; p <= PointsPerDevice; p++ {
			pointID := generateID()
			nodeStmt.Exec(pointID, "point", deviceID, fmt.Sprintf("Point-%s-%d", deviceID[:8], p), fmt.Sprintf("Point %d", p))

			// Create ports
			inputPortID := generateID()
			outputPortID := generateID()
			portStmt.Exec(inputPortID, pointID, "input", fmt.Sprintf("Input-%s", pointID[:8]), "Input port")
			portStmt.Exec(outputPortID, pointID, "output", fmt.Sprintf("Output-%s", pointID[:8]), "Output port")

			// Create port values (fewer than SQLite version to save time)
			numValues := rand.Intn(3) + 1
			for v := 0; v < numValues; v++ {
				valueID := generateID()
				timestamp := time.Now().Add(-time.Duration(rand.Intn(86400)) * time.Second)
				portValueStmt.Exec(valueID, inputPortID, timestamp, rand.Float64()*100, fmt.Sprintf("value-%s", valueID[:8]), rand.Intn(2) == 1, rand.Intn(2) == 1)
			}

			// Create tags (10-25 tags per point for rich metadata)
			numTags := rand.Intn(16) + 10
			for t := 0; t < numTags; t++ {
				tagStmt.Exec(pointID, tagKeys[rand.Intn(len(tagKeys))], tagValues[rand.Intn(len(tagValues))])
			}
		}

		nodeStmt.Close()
		portStmt.Close()
		portValueStmt.Close()
		tagStmt.Close()
		tx.Commit()

		processedCount++
		if processedCount%100 == 0 {
			log.Printf("Worker %d: Processed %d devices", workerID, processedCount)
		}
	}
}

func createEdges(db *sql.DB) {
	// Sample a subset of ports to create edges (creating edges for 250M nodes would take too long)
	log.Println("Creating sample edges...")

	tx, _ := db.Begin()
	defer tx.Rollback()

	// Create ~100k edges
	edgeStmt, _ := tx.Prepare("INSERT INTO edges (id, from_port_id, to_port_id, description) VALUES ($1, $2, $3, $4)")
	defer edgeStmt.Close()

	rows, _ := db.Query("SELECT id FROM ports WHERE port_type = 'output' ORDER BY RANDOM() LIMIT 50000")
	outputPorts := make([]string, 0)
	for rows.Next() {
		var portID string
		rows.Scan(&portID)
		outputPorts = append(outputPorts, portID)
	}
	rows.Close()

	rows, _ = db.Query("SELECT id FROM ports WHERE port_type = 'input' ORDER BY RANDOM() LIMIT 50000")
	inputPorts := make([]string, 0)
	for rows.Next() {
		var portID string
		rows.Scan(&portID)
		inputPorts = append(inputPorts, portID)
	}
	rows.Close()

	for i := 0; i < len(outputPorts) && i < len(inputPorts); i++ {
		edgeID := generateID()
		edgeStmt.Exec(edgeID, outputPorts[i], inputPorts[i], fmt.Sprintf("Edge %d", i+1))

		if (i+1)%10000 == 0 {
			log.Printf("Created %d edges", i+1)
		}
	}

	tx.Commit()
	log.Println("Edges created")
}

func analyzeTables(db *sql.DB) {
	tables := []string{"nodes", "ports", "port_values", "edges", "tags"}
	for _, table := range tables {
		log.Printf("Analyzing table: %s", table)
		db.Exec(fmt.Sprintf("ANALYZE %s", table))
	}
	log.Println("Analysis complete")
}
