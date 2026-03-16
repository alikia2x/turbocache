FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o turbocache .

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/turbocache .
COPY --from=builder /app/docs ./docs

EXPOSE 3000

ENV CACHE_DIRECTORY=/app/cache
ENV PORT=3000

RUN mkdir -p /app/cache

ENTRYPOINT ["./turbocache"]
