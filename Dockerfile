# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary with stripped debug info for minimal size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o finsec-web ./cmd/finsec-web

# Final lightweight runtime stage (~15 MB)
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/finsec-web /app/finsec-web
COPY --from=builder /app/config.example.yaml /app/config.example.yaml

# Create start script that copies config.example.yaml to config.yaml if config.yaml doesn't exist
RUN echo '#!/bin/sh' > /app/entrypoint.sh && \
    echo 'if [ ! -f /app/config.yaml ]; then' >> /app/entrypoint.sh && \
    echo '  cp /app/config.example.yaml /app/config.yaml' >> /app/entrypoint.sh && \
    echo 'fi' >> /app/entrypoint.sh && \
    echo 'exec /app/finsec-web -open=false' >> /app/entrypoint.sh && \
    chmod +x /app/entrypoint.sh

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
