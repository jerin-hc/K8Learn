# ---- Build stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/server .

# ---- Run stage ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 8080

# ENV POSTGRES_HOST=localhost
# ENV POSTGRES_PORT=5432
# ENV POSTGRES_USER=postgres
# ENV POSTGRES_PASSWORD=postgres
# ENV POSTGRES_NAME=k8learn
ENV POSTGRES_SSLMODE=disable
# ENV PORT=8080

CMD ["./server"]
