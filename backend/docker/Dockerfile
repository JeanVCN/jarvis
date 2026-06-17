FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /jarvis-api ./cmd/api

FROM alpine:3.20

RUN addgroup -S jarvis && adduser -S jarvis -G jarvis

WORKDIR /app
COPY --from=builder /jarvis-api /app/jarvis-api

USER jarvis
EXPOSE 18080

CMD ["/app/jarvis-api"]
