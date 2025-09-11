FROM golang:1.24-alpine AS builder
WORKDIR /app

# TODO: No vendor deps yet...
# COPY go.mod go.sum ./
# RUN go mod download
COPY . .

RUN GOOS=linux go build -o scrollwork cmd/scrollwork/main.go

FROM alpine:3.22.1

WORKDIR /root/
COPY --from=builder /app/scrollwork .
COPY --from=builder /app/cmd/scrollwork/banner.txt .

RUN mkdir -p /tmp

CMD ["./scrollwork"]
