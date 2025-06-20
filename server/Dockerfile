# Use official Go image as the base
FROM golang:1.24.3-alpine

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum from the server directory
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code from the server directory
COPY . .

# Build the Go application
RUN go build -o url-shortener .

# Expose port 8080
EXPOSE 8080

# Command to run the application
CMD ["./url-shortener"]