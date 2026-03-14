FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY notification-service/go.mod notification-service/go.sum ./notification-service/

WORKDIR /app/notification-service
RUN go mod download

COPY notification-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service ./cmd/main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/notification-service/notification-service .
RUN chmod +x ./notification-service

CMD ["./notification-service"]