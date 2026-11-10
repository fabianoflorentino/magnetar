# Build stage
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /magnetar ./cmd/magnetar

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /magnetar .

# GOMEMLIMIT previne OOM do GC em contêineres (golang-performance).
ENV GOMEMLIMIT=1GiB

EXPOSE 8080

ENTRYPOINT ["./magnetar"]
