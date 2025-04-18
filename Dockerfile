FROM golang:1.23.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# static the go app
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    sqlite3 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/main .

COPY static/ /app/static/

RUN mkdir -p /app/data

ENV DB_PATH=/app/data/main.db

EXPOSE 8090

CMD ["./main"]