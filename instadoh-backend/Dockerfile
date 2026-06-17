FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o instadoh .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -g '' instadoh
USER instadoh

WORKDIR /app

COPY --from=builder /app/instadoh .

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./instadoh"]