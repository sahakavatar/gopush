# WebSocket Server with Redis and TLS Configuration

## Overview

This document outlines the setup for the WebSocket server using Redis and optional TLS. By default, TLS is disabled, and you can enable it if needed.

## Configuration File Example

Below is the example configuration file (`config.json`):

```json
{
   "redis": {
      "nodes": [
         {
            "address": "127.0.0.1:6379", // Change with your Redis host
            "password": null
         }
      ],
      "channels_pattern": "test-*"
   },
   "server": {
      "host": "0.0.0.0:9000", // Change with your WebSocket server host
      "port": "6001",
      "protocol": "ws", // Use 'wss' if working on SSL
      "ws_url": "/ws",
      "tls": {
         "Enabled": false, // TLS is disabled by default
         "cert_file": "/path/to/your_file.pem", // Path to your TLS certificate file (optional)
         "key_file": "/path/to/your_file.pem" // Path to your TLS private key (optional)
      },
      "authorize": {
         "url": "http://your-domain/verify-token", // Authorization token verification URL
         "timeout": 5000,
         "cache_time_out": 3600
      },
      "health_check_url": "/health" // Health check endpoint URL
   },
   "logging": {
      "level": "info",
      "file": "/var/log/websocket-server.log" // Log file path
   },
   "environment": "locale" // Set the environment (e.g., 'production', 'development')
}
```

## Dependencies

- Go 1.18+
- `github.com/go-redis/redis/v8` - Redis client for Go
- `github.com/gorilla/websocket` - WebSocket client/server for Go
- `golang.org/x/net/context` - Context package for Go

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/sahakavatar/gopush.git
   cd websocket-server
   ```

2. Install dependencies:

   ```bash
   go mod tidy
   ```

3. Update the `config.json` file with your configuration values.

4. Run the server:

   ```bash
   go run main.go
   ```

## WebSocket API

### Subscribe to a channel

```json
{
  "action": "subscribe",
  "token": "your-token-here",
  "channel": "test-channel"
}
```

### Send a message

```json
{
  "action": "send",
  "channel": "test-channel",
  "message": "Hello, Redis!"
}
```

## Logging

Logs are written to a file (`/var/log/websocket-server.log` by default) or to the standard output (if the environment is not production). The log level can be configured in the `config.json` file.

## Health Check

Access the health check URL:

```
/health
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Example: Subscribe with React and JavaScript

Here’s a simple example to subscribe to a WebSocket channel using React and JavaScript.

```javascript
import React, { useEffect } from 'react';

const WebSocketComponent = () => {
  useEffect(() => {
    const socket = new WebSocket('ws://your-websocket-server/ws');

    socket.onopen = () => {
      console.log('WebSocket connected');
      socket.send(JSON.stringify({
        action: "subscribe",
        token: "your-token-here",
        channel: "test-channel"
      }));
    };

    socket.onmessage = (event) => {
      console.log('Message received:', event.data);
    };

    socket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    socket.onclose = () => {
      console.log('WebSocket closed');
    };

    // Clean up the connection when the component is unmounted
    return () => {
      socket.close();
    };
  }, []);

  return (
    <div>
      <h1>WebSocket Example</h1>
      <p>Check the console for WebSocket messages.</p>
    </div>
  );
};

export default WebSocketComponent;
```

This code demonstrates how to connect to a WebSocket server, subscribe to a channel, and handle incoming messages with React and JavaScript.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

