package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
	"socket/auth"
	"socket/config"
)

// SubscriptionMessage represents the structure sent to clients
type SubscriptionMessage struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Channel   string `json:"channel"`
	Event     string `json:"event"`
	WsUrl     string `json:"ws_url"`
	ExpiresAt int64  `json:"expires_at"`
}

var mu sync.Mutex
var clients = make(map[*websocket.Conn]string)

// HandleSubscribe handles WebSocket subscription requests
// Now accepting a slice of Redis clients (rdbs)
func HandleSubscribe(rdbs []*redis.Client, conn *websocket.Conn, data map[string]interface{}, config *config.Config) {
	token, ok := data["token"].(string)
	if !ok {
		SendMessageToClient(conn, "Invalid or missing token")
		log.Printf("Received invalid or missing token from client: %v", conn.RemoteAddr())
		return
	}

	authorizeURL := config.Server.Authorize.Url
	isValid, err := auth.ValidateToken(rdbs[0], token, authorizeURL, config.Server.Authorize.CashTimeOut) // Assuming using the first client for token validation
	if err != nil || !isValid {
		SendMessageToClient(conn, "Token validation failed")
		log.Printf("Token validation failed for client %v with token %s: %v", conn.RemoteAddr(), token, err)
		return
	}

	channel, ok := data["channel"].(string)
	if !ok {
		SendMessageToClient(conn, "Channel not specified")
		log.Printf("Channel not specified in subscription request from client %v", conn.RemoteAddr())
		return
	}

	mu.Lock()
	clients[conn] = channel
	mu.Unlock()

	// You can implement logic to choose the appropriate Redis client from the list (rdbs)
	// For example, you can select based on channel name or some other logic
	selectedClient := rdbs[0] // Selecting the first Redis client (for now)

	// Start listening to the Redis channel asynchronously
	go SubscribeToRedisChannel(selectedClient, conn, channel)

	expiration := time.Now().Add(time.Duration(config.Server.Authorize.CashTimeOut) * time.Minute).Unix()
	subscriptionMessage := SubscriptionMessage{
		Status:    "success",
		Message:   fmt.Sprintf("Subscribed to channel: %s", channel),
		Channel:   channel,
		Event:     "subscription",
		WsUrl:     fmt.Sprintf("ws://%s:%s%s", config.Server.Host, config.Server.Port, config.Server.WsUrl),
		ExpiresAt: expiration,
	}
	SendMessageToClient(conn, MarshalMessage(subscriptionMessage))

	log.Printf("Client %v successfully subscribed to channel %s", conn.RemoteAddr(), channel)
}

// SubscribeToRedisChannel listens for messages on a Redis channel
func SubscribeToRedisChannel(rdb *redis.Client, conn *websocket.Conn, channel string) {
	pubsub := rdb.Subscribe(context.Background(), channel)
	defer pubsub.Close()

	log.Printf("Listening for messages on channel %s", channel)

	for msg := range pubsub.Channel() {
		log.Printf("Received message on channel %s: %s", channel, msg.Payload)
		SendMessageToClient(conn, msg.Payload)
	}

	mu.Lock()
	delete(clients, conn)
	mu.Unlock()

	log.Printf("Client %v unsubscribed from channel %s", conn.RemoteAddr(), channel)
}

// SendMessageToClient sends a message to a WebSocket client
func SendMessageToClient(conn *websocket.Conn, message string) {
	err := conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Printf("Failed to send WebSocket message to client %v: %v", conn.RemoteAddr(), err)
	}
}

// MarshalMessage converts a message to JSON
func MarshalMessage(message SubscriptionMessage) string {
	bytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return ""
	}
	return string(bytes)
}
