# Stage 1: Build the Go app
FROM golang:1.24.1-bookworm AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o OJ_API .

# Stage 2: Create a smaller image for running the Go app
FROM debian:bookworm

# Install isolate from the official repository
RUN apt-get update && \
    apt-get install -y --no-install-recommends git pkg-config libcap-dev libsystemd-dev ca-certificates make gcc g++ cmake python3 python3-pip python3-venv ninja-build libgtest-dev valgrind && \
    git clone https://github.com/ioi/isolate.git /isolate && \
    cd /isolate && \
    make install && \
    rm -rf /isolate && \
    mkdir -p /sandbox /sandbox/code /sandbox/repo && \
    chmod 777 /sandbox /sandbox/code /sandbox/repo && \
    apt-get remove -y git pkg-config libcap-dev libsystemd-dev && \
    apt-get autoremove -y && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN pip install art --break-system-packages

RUN mkdir -p /app
WORKDIR /app

# Copy the built Go app from the builder stage
COPY --from=builder /app/.env.local /app/.env.local
COPY --from=builder /app/sandbox/python /app/sandbox/python
COPY --from=builder /app/OJ_API /app/OJ_API

RUN chmod +x /app/OJ_API

# Expose port 3001 to the outside world
EXPOSE 3001

# Command to run the executable
ENTRYPOINT ["/app/OJ_API"]