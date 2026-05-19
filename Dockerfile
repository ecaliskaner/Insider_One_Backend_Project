FROM golang:1.26.3-alpine3.22 AS builder

# Install GCC and musl-dev for SQLite CGO compilation
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Run stage
FROM alpine:3.22

# Install tzdata and ca-certificates
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable (stitching migration, seeding, and serving for zero-friction DX)
CMD ["/bin/sh", "-c", "./main migrate up && ./main seed && ./main serve"]
