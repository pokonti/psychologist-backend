# ===== Build Stage =====
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY proto ./proto

COPY user-service/go.mod user-service/go.sum ./user-service/

WORKDIR /app/user-service
RUN go mod download

COPY user-service/ ./

RUN GOOS=linux GOARCH=arm64 go build -o user-service ./cmd/main.go

# ===== Runtime Stage =====
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/user-service/user-service ./user-service
RUN chmod +x ./user-service

EXPOSE 8081 9091
CMD ["./user-service"]

