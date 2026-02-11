FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY proto ./proto

COPY gateway/go.mod gateway/go.sum ./gateway/

WORKDIR /app/gateway
RUN go mod download

COPY gateway/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o gateway ./cmd/main.go

# ===== Runtime Stage =====
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/gateway/gateway .

RUN chmod +x ./gateway

EXPOSE 8080 9090

CMD ["./gateway"]