FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY proto ./proto

COPY auth-service/go.mod auth-service/go.sum ./auth-service/

WORKDIR /app/auth-service
RUN go mod download

COPY auth-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/main.go

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/auth-service/auth-service .

RUN chmod +x ./auth-service

EXPOSE 8083 9093

CMD ["./auth-service"]

