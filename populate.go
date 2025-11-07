package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	NetworkCount      = 50
	DevicesPerNetwork = 50
	PointsPerDevice   = 100
	PortsPerNode      = 2 // input and output ports per node
)

var (
	tagKeys   = []string{"location", "category", "priority", "status", "environment", "region", "zone", "criticality"}
	tagValues = []string{"production", "staging", "development", "high", "medium", "low", "active", "inactive", "north", "south", "east", "west", "zone-a", "zone-b", "zone-c"}
)

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

	// Connect to the SQLite database in the Docker volume
	// You can also pass the database path as a command line argument
	dbPath := "./rubix.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal(err)
	}

	// Start transaction for better performance
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	log.Println("Starting data population...")
	startTime := time.Now()

	// Prepare statements
	nodeStmt, err := tx.Prepare("INSERT INTO nodes (id, type, parent_id, name, description) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer nodeStmt.Close()

	portStmt, err := tx.Prepare("INSERT INTO ports (id, node_id, port_type, name, description) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer portStmt.Close()

	portValueStmt, err := tx.Prepare("INSERT INTO port_values (id, port_id, timestamp, value_numeric, value_text, value_boolean, is_synced) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer portValueStmt.Close()

	tagStmt, err := tx.Prepare("INSERT OR IGNORE INTO tags (node_id, tag_key, tag_value) VALUES (?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer tagStmt.Close()

	edgeStmt, err := tx.Prepare("INSERT INTO edges (id, from_port_id, to_port_id, description) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer edgeStmt.Close()

	totalNodes := 0
	totalPorts := 0
	totalPortValues := 0
	totalTags := 0
	totalEdges := 0
	allPorts := make([]string, 0, 100000) // Store port IDs for creating edges

	// Create networks
	log.Println("Creating networks...")
	for i := 1; i <= NetworkCount; i++ {
		networkID := generateID()
		_, err := nodeStmt.Exec(networkID, "network", nil, fmt.Sprintf("Network-%d", i), fmt.Sprintf("Network %d description", i))
		if err != nil {
			log.Fatal(err)
		}
		totalNodes++

		// Add tags to network (0-4 tags)
		numTags := rand.Intn(5)
		for t := 0; t < numTags; t++ {
			key := tagKeys[rand.Intn(len(tagKeys))]
			value := tagValues[rand.Intn(len(tagValues))]
			_, _ = tagStmt.Exec(networkID, key, value)
			totalTags++
		}

		// Create ports for network
		networkPorts := createPorts(portStmt, networkID, &totalPorts, &allPorts)

		// Create devices under each network
		log.Printf("Creating devices for Network-%d...", i)
		for d := 1; d <= DevicesPerNetwork; d++ {
			deviceID := generateID()
			_, err := nodeStmt.Exec(deviceID, "device", networkID, fmt.Sprintf("Device-%d-%d", i, d), fmt.Sprintf("Device %d of Network %d", d, i))
			if err != nil {
				log.Fatal(err)
			}
			totalNodes++

			// Add tags to device (0-4 tags)
			numTags := rand.Intn(5)
			for t := 0; t < numTags; t++ {
				key := tagKeys[rand.Intn(len(tagKeys))]
				value := tagValues[rand.Intn(len(tagValues))]
				_, _ = tagStmt.Exec(deviceID, key, value)
				totalTags++
			}

			// Create ports for device
			devicePorts := createPorts(portStmt, deviceID, &totalPorts, &allPorts)

			// Create points under each device
			for p := 1; p <= PointsPerDevice; p++ {
				pointID := generateID()
				_, err := nodeStmt.Exec(pointID, "point", deviceID, fmt.Sprintf("Point-%d-%d-%d", i, d, p), fmt.Sprintf("Point %d of Device %d", p, d))
				if err != nil {
					log.Fatal(err)
				}
				totalNodes++

				// Add tags to point (0-4 tags)
				numTags := rand.Intn(5)
				for t := 0; t < numTags; t++ {
					key := tagKeys[rand.Intn(len(tagKeys))]
					value := tagValues[rand.Intn(len(tagValues))]
					_, _ = tagStmt.Exec(pointID, key, value)
					totalTags++
				}

				// Create ports for point
				pointPorts := createPorts(portStmt, pointID, &totalPorts, &allPorts)

				// Create port values for point ports
				for _, portID := range pointPorts {
					// Create 1-5 values per port
					numValues := rand.Intn(5) + 1
					for v := 0; v < numValues; v++ {
						valueID := generateID()
						timestamp := time.Now().Add(-time.Duration(rand.Intn(86400)) * time.Second)
						_, err := portValueStmt.Exec(
							valueID,
							portID,
							timestamp,
							rand.Float64()*100,
							fmt.Sprintf("value-%s", valueID[:8]),
							rand.Intn(2) == 1,
							rand.Intn(2) == 1,
						)
						if err != nil {
							log.Fatal(err)
						}
						totalPortValues++
					}
				}
			}

			// Create edges between device ports and some point ports
			if len(devicePorts) > 0 && d > 1 {
				// Create 1-3 edges per device
				numEdges := rand.Intn(3) + 1
				for e := 0; e < numEdges && len(allPorts) > 10; e++ {
					fromPort := devicePorts[rand.Intn(len(devicePorts))]
					toPort := allPorts[rand.Intn(len(allPorts))]
					if fromPort != toPort {
						edgeID := generateID()
						_, _ = edgeStmt.Exec(edgeID, fromPort, toPort, fmt.Sprintf("Edge from device %d", d))
						totalEdges++
					}
				}
			}
		}

		// Create edges between network ports
		if i > 1 && len(networkPorts) > 0 && len(allPorts) > 100 {
			numEdges := rand.Intn(5) + 1
			for e := 0; e < numEdges; e++ {
				fromPort := networkPorts[rand.Intn(len(networkPorts))]
				toPort := allPorts[rand.Intn(len(allPorts))]
				if fromPort != toPort {
					edgeID := generateID()
					_, _ = edgeStmt.Exec(edgeID, fromPort, toPort, fmt.Sprintf("Edge from network %d", i))
					totalEdges++
				}
			}
		}

		log.Printf("Completed Network-%d: Total nodes so far: %d", i, totalNodes)
	}

	// Commit transaction
	log.Println("Committing transaction...")
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	duration := time.Since(startTime)

	log.Println("\n=== Data Population Complete ===")
	log.Printf("Total Nodes: %d", totalNodes)
	log.Printf("Total Ports: %d", totalPorts)
	log.Printf("Total Port Values: %d", totalPortValues)
	log.Printf("Total Tags: %d", totalTags)
	log.Printf("Total Edges: %d", totalEdges)
	log.Printf("Time taken: %s", duration)
}

func createPorts(stmt *sql.Stmt, nodeID string, totalPorts *int, allPorts *[]string) []string {
	ports := make([]string, 0, PortsPerNode)

	// Create input port
	inputPortID := generateID()
	_, err := stmt.Exec(inputPortID, nodeID, "input", fmt.Sprintf("Input-%s", nodeID[:8]), "Input port")
	if err != nil {
		log.Fatal(err)
	}
	*totalPorts++
	ports = append(ports, inputPortID)
	*allPorts = append(*allPorts, inputPortID)

	// Create output port
	outputPortID := generateID()
	_, err = stmt.Exec(outputPortID, nodeID, "output", fmt.Sprintf("Output-%s", nodeID[:8]), "Output port")
	if err != nil {
		log.Fatal(err)
	}
	*totalPorts++
	ports = append(ports, outputPortID)
	*allPorts = append(*allPorts, outputPortID)

	return ports
}
