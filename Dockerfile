FROM golang:1.22-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code and templates
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary and template from builder
COPY --from=builder /app/server .
COPY --from=builder /app/index.html .

EXPOSE 8080

CMD ["./server"]
