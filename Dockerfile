# Stage 1: Build the Go app
FROM golang:1.24.5-alpine3.22 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Set GIN_MODE to release
ENV GIN_MODE=release

# Build the Go app
RUN go build -o OJ_API .

# Stage 2: Create a smaller image for running the Go app
FROM alpine:3.22

RUN mkdir -p /app
WORKDIR /app

# Copy the built Go app from the builder stage
# COPY --from=builder /app/.env.local /app/.env.local
RUN touch ./app/.env.local
COPY --from=builder /app/OJ_API /app/OJ_API

RUN chmod +x /app/OJ_API

# Expose port 3001 to the outside world
EXPOSE 3001

# Command to run the executable
ENTRYPOINT ["/app/OJ_API"]