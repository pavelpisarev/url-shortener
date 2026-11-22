# Build stage
FROM golang:1.27-alpine AS builder

# Set the working directory
WORKDIR /app

# Build a standalone static binary for Linux (amd64) to run in a minimal container
ENV CGO_ENABLED=0

# Copy and download dependencies for backend
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate


# Final stage
FROM alpine:3.24

# Set the working directory
WORKDIR /app

# Install necessary packages and create a non-root user
RUN apk add --no-cache ca-certificates \
 && addgroup -S appgroup \
 && adduser -S appuser -G appgroup

# Copy the built binary from the builder stage
COPY --from=builder --chown=appuser:appgroup /out/api /app/api
COPY --from=builder --chown=appuser:appgroup /out/migrate /app/migrate

# Set the user to run the application
USER appuser

# Expose the port the application will run on
EXPOSE 8080

# Run the application
CMD ["/app/api"]