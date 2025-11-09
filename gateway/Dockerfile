# ===== Build Stage =====
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy modules and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary for linux/arm64 (needed for M1/M2 Mac)
RUN GOOS=linux GOARCH=arm64 go build -o gateway ./main.go

# ===== Runtime Stage =====
FROM alpine:3.19

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gateway ./gateway

# Ensure it is executable
RUN chmod +x ./gateway

# Expose service ports
EXPOSE 8080 9090

# Run the service
CMD ["./gateway"]

