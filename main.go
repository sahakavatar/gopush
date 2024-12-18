package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	gws "github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"socket/auth"
	"socket/config"
	"socket/websocket"
)

// setupLogging sets up logging, creating the log file if necessary
func setupLogging(config *config.Config) (*os.File, error) {
	// Check if the environment is production
	if config.Environment == "production" {
		logFilePath := "/var/log/websocket-server.log"

		// Check if the log file exists
		if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
			// Create the file if it doesn't exist
			file, err := os.Create(logFilePath)
			if err != nil {
				return nil, fmt.Errorf("Failed to create log file: %v", err)
			}
			log.Printf("Log file created: %s", logFilePath)
			return file, nil
		}

		// If the file exists, open it
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("Failed to open log file: %v", err)
		}
		return file, nil
	}

	// If not production, log to stdout
	return os.Stdout, nil
}

func main() {

	// Initialize the logger
	logger, err := auth.InitLogger("your_log_file.log")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Assign the global logger
	auth.SetLogger(logger)
	// Load the configuration
	config, err := config.LoadConfig("/app/config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up logging
	logFile, err := setupLogging(config)
	if err != nil {
		log.Fatalf("Error setting up logging: %v", err)
	}
	defer logFile.Close()

	// Set up the logger to write to the file
	log.SetOutput(logFile)

	// Initialize Redis cluster clients for each node with individual passwords
	var rdbs []*redis.Client
	for _, node := range config.Redis.Nodes {
		client := redis.NewClient(&redis.Options{
			Addr:     node.Address,
			Password: node.Password, // Password for each Redis node
		})

		// Health check to ensure the connection is alive
		_, err := client.Ping(context.Background()).Result()
		if err != nil {
			log.Fatalf("Failed to connect to Redis node %s: %v", node.Address, err)
		}
		rdbs = append(rdbs, client)
	}

	// WebSocket server setup
	http.HandleFunc(config.Server.WsUrl, func(w http.ResponseWriter, r *http.Request) {
		upgrader := &gws.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		log.Printf("New WebSocket connection from %s", r.RemoteAddr)

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read failed: %v", err)
				break
			}

			var data map[string]interface{}
			if err := json.Unmarshal(message, &data); err != nil {
				websocket.SendMessageToClient(conn, "Invalid message format")
				continue
			}

			action, ok := data["action"].(string)
			if !ok {
				websocket.SendMessageToClient(conn, "Action not specified")
				continue
			}

			if action == "subscribe" {
				websocket.HandleSubscribe(rdbs, conn, data, config)
			} else if action == "send" {
				handleSend(rdbs, conn, data)
			}
		}
	})

	// Check if TLS is enabled (wss://)
	if config.Server.TLS.Enabled {
		// Ensure cert and key files exist for TLS
		certFile := config.Server.TLS.CertFile
		keyFile := config.Server.TLS.KeyFile
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			log.Fatalf("TLS cert file not found: %v", err)
		}
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			log.Fatalf("TLS key file not found: %v", err)
		}

		// Custom TLS configuration to allow self-signed certs for testing
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // Disable verification for self-signed certificates (for testing)
		}

		// Start the secure WebSocket server (wss://)
		address := fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port)
		log.Printf("WebSocket server started at wss://%s", address)

		// Create custom TLS listener
		server := &http.Server{
			Addr:      address,
			TLSConfig: tlsConfig,
		}

		log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
	} else {
		// Start the non-secure WebSocket server (ws://)
		address := fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port)
		log.Printf("WebSocket server started at ws://%s", address)
		log.Fatal(http.ListenAndServe(address, nil))
	}
}

func handleSend(rdbs []*redis.Client, conn *gws.Conn, data map[string]interface{}) {
	channel, ok := data["channel"].(string)
	if !ok {
		websocket.SendMessageToClient(conn, "Channel not specified")
		return
	}

	message, err := json.Marshal(data)
	if err != nil {
		websocket.SendMessageToClient(conn, "Invalid message format")
		return
	}

	// Publish to all Redis nodes (could be optimized if only a specific node should be targeted)
	var publishErr error
	for _, rdb := range rdbs {
		err = rdb.Publish(context.Background(), channel, message).Err()
		if err != nil {
			publishErr = err
			log.Printf("Failed to publish message to Redis node: %v", err)
		}
	}

	if publishErr != nil {
		websocket.SendMessageToClient(conn, "Failed to publish message")
		return
	}

	websocket.SendMessageToClient(conn, "Message sent successfully")
}
