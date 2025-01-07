FROM golang:1.23-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy rest of the application
COPY . .

# Expose port
EXPOSE 8080