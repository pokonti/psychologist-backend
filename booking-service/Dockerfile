# ===== Build Stage =====
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY proto ./proto

COPY user-service/go.mod user-service/go.sum ./user-service/

WORKDIR /app/booking-service
RUN go mod download

COPY booking-service/ ./

RUN GOOS=linux GOARCH=arm64 go build -o booking-service ./cmd/main.go

# ===== Runtime Stage =====
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/booking-service/booking-service ./booking-service
RUN chmod +x ./booking-service

EXPOSE 8084 9094
CMD ["./user-service"]

