# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o termviewer-backend ./main.go

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install root certificates for HTTPS calls
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy binary from builder
COPY --from=builder /app/termviewer-backend .

# Use non-root user
USER appuser:appgroup

EXPOSE 3001

CMD ["./termviewer-backend"]
