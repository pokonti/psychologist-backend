FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY proto ./proto

COPY booking-service/go.mod booking-service/go.sum ./booking-service/

WORKDIR /app/booking-service
RUN go mod download

COPY booking-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o booking-service ./cmd/main.go

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/booking-service/booking-service .
RUN chmod +x ./booking-service

EXPOSE 8084 9094

CMD ["./booking-service"]

