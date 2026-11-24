# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/gopherwallet ./cmd/api

# Run stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/gopherwallet /app/gopherwallet
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080

CMD ["/app/gopherwallet"]
