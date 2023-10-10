# Use an official Go runtime as a parent image for the build stage
FROM golang:1.21 AS build

# Set the working directory in the container
WORKDIR /go/src/app

# Copy the local package files to the container's workspace
COPY . .

# Build the application, producing a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -v -o app

# Use Debian 12 slim for the final stage
FROM debian:12-slim

# Copy the binary from the build stage to the final stage
COPY --from=build /go/src/app/app /app

# Ensure you have the necessary certificates to make HTTPS connections
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# This command runs the application
CMD ["/app"]

