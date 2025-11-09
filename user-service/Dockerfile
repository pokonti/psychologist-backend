FROM golang:1.24-alpine

WORKDIR /app

# Install necessary packages for build (optional)
RUN apk add --no-cache git bash

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary
RUN go build -o user-service main.go

# Make sure the binary is executable
RUN chmod +x user-service

EXPOSE 8001

CMD ["./user-service"]

