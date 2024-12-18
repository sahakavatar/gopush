# Use the official Golang image as a build stage
FROM golang:1.23.2 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files into the container
COPY go.mod go.sum ./

# Copy the config.json file into the container
COPY config.json ./config.json

# Create a directory for SSL certificates inside /app
RUN mkdir -p /app/ssl

# Copy SSL certificate and private key into the container (into /app/ssl)
COPY /ssl/fullchain.pem /app/ssl/fullchain.pem
COPY /ssl/privkey.pem /app/ssl/privkey.pem

# Download the module dependencies
RUN go mod download

# Copy the source code into the container
COPY . ./

# Build the Go application with CGO enabled for Kafka support
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -o main .

# Use an Ubuntu-based image for the final stage to ensure compatibility with CGO
FROM ubuntu:22.04

# Install necessary certificates and dependencies for CGO (like libc)
RUN apt-get update && apt-get install -y ca-certificates libc6

# Set the working directory in the final image
WORKDIR /root/

# Copy the compiled Go binary from the builder stage
COPY --from=builder /app/main .

# Copy config.json and SSL certificates from the builder stage to the final image
COPY --from=builder /app/config.json /app/config.json
COPY --from=builder /app/ssl /app/ssl

# Ensure the binary is executable
RUN chmod +x /root/main

# Expose the port your app runs on
EXPOSE 6001

# Command to run the executable
CMD ["./main"]
