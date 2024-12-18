package auth

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

// Global logger variable
var logger *log.Logger

// SetLogger sets the global logger instance
func SetLogger(l *log.Logger) {
	logger = l
}

// InitLogger initializes the logger based on the log file or standard output
func InitLogger(logFile string) (*log.Logger, error) {
	var logOutput *os.File
	var err error

	if logFile != "" {
		// Log to file
		logOutput, err = os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return nil, fmt.Errorf("could not open log file: %v", err)
		}
	} else {
		// Log to standard output
		logOutput = os.Stdout
	}

	// Initialize the logger with INFO prefix and appropriate flags for date, time, and file
	logger = log.New(logOutput, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	return logger, nil
}

// ValidateToken validates a token using Redis and an external API
func ValidateToken(rdb *redis.Client, token, authorizeURL string, cacheTimeout int16) (bool, error) {
	ctx := context.Background()

	// Check if logger is initialized
	if logger == nil {
		return false, fmt.Errorf("logger is not initialized")
	}

	// Log the start of the token validation
	logger.Printf("Validating token: %s", token)

	// Check the cache for the token first
	cached, err := rdb.Get(ctx, token).Result()
	if err == redis.Nil {
		// Token is not found in cache, so we call the external API
		logger.Printf("Token %s not found in cache. Calling authorization API...", token)

		isValid, err := CallAuthorizeAPI(token, authorizeURL)
		if err != nil {
			logger.Printf("Authorization API call failed for token %s: %v", token, err)
			return false, fmt.Errorf("authorization API call failed: %v", err)
		}

		// Cache the result of the validation
		ttl := time.Duration(cacheTimeout) * time.Minute
		if isValid {
			rdb.Set(ctx, token, "valid", ttl)
			logger.Printf("Token %s is valid. Cached with TTL %d minutes.", token, cacheTimeout)
		} else {
			rdb.Set(ctx, token, "invalid", ttl)
			logger.Printf("Token %s is invalid. Cached with TTL %d minutes.", token, cacheTimeout)
		}
		return isValid, nil
	} else if err != nil {
		// Error occurred while fetching the token from Redis
		logger.Printf("Error fetching token %s from Redis: %v", token, err)
		return false, fmt.Errorf("error fetching token from Redis: %v", err)
	}

	// If the token is found in cache, log the result
	if cached == "valid" {
		logger.Printf("Token %s is valid (cached).", token)
	} else {
		logger.Printf("Token %s is invalid (cached).", token)
	}

	return cached == "valid", nil
}

// CallAuthorizeAPI makes a request to the authorization API to validate the token
func CallAuthorizeAPI(token, authorizeURL string) (bool, error) {
	logger.Printf("Calling authorization API for token: %s", token)

	req, err := http.NewRequest("POST", authorizeURL, nil)
	if err != nil {
		// Log the failure to create the HTTP request
		logger.Printf("Failed to create request for token %s: %v", token, err)
		return false, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		// Log the failure of the API request
		logger.Printf("API request failed for token %s: %v", token, err)
		return false, fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for detailed error logging using io.ReadAll
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("Failed to read response body for token %s: %v", token, err)
	}

	// Log the response body for debugging
	logger.Printf("API Response for token %s: %s", token, string(body))

	// Check the response from the authorization API
	if resp.StatusCode == http.StatusOK {
		logger.Printf("Authorization API for token %s returned OK", token)
		return true, nil
	}

	logger.Printf("Authorization API for token %s returned non-OK status: %d", token, resp.StatusCode)
	return false, nil
}
