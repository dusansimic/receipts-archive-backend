# Start from 1.14 alpine image
FROM golang:1.14-alpine as builder

RUN apk --no-cache add make build-base

# Maintainer info
LABEL maintainer="Dušan Simić <dusan.simic1810@gmail.com>"

# Set working dir inside the container
WORKDIR /app

# Copy mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the app
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags -static' -o main .

# Start new stage
FROM alpine

WORKDIR /app

COPY --from=builder /app/main .

# Setup environment variables
ENV PORT=3000

EXPOSE ${PORT}

# Run the backend
CMD ["./main"]
