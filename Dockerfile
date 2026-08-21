# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
COPY migrations/ ./migrations/

EXPOSE 8081

CMD ["./server"]
