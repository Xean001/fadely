# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for updates if needed
RUN apk add --no-cache git

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# CGO_ENABLED=0 for a static binary (if possible), but we might need CGO for some libs?
# kkdai/youtube doesn't strictly need CGO.
# Using CGO_ENABLED=0 is safer for alpine compat.
RUN CGO_ENABLED=0 GOOS=linux go build -o downtube ./cmd/server/main.go

# Run Stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies: FFmpeg is crucial
RUN apk add --no-cache ffmpeg ca-certificates mailcap

# Copy the binary from the builder stage
COPY --from=builder /app/downtube .

# Copy static assets (HTML, CSS, JS)
COPY --from=builder /app/static ./static

# Expose the application port
EXPOSE 8081

# Command to run the executable
CMD ["./downtube"]
