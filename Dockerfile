#builder
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ratelimiter ./cmd/ratelimiter

# runner
FROM alpine:3.21

RUN adduser -D -u 10001 app

COPY --from=builder /out/ratelimiter /usr/local/bin/ratelimiter

USER app
EXPOSE 8080

ENTRYPOINT ["ratelimiter"]
