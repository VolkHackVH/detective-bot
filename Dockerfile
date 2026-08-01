FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/detective-bot \
    ./cmd/bot

FROM alpine:3.22

RUN apk add --no-cache ca-certificates && \
    addgroup -S bot && \
    adduser -S bot -G bot && \
    mkdir -p /app/data && \
    chown -R bot:bot /app

WORKDIR /app

COPY --from=builder /out/detective-bot /app/detective-bot

USER bot

VOLUME ["/app/data"]

ENTRYPOINT ["/app/detective-bot"]