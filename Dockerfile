# build stage
FROM golang:1.23.1-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN go build -o loggerator ./cmd/main.go

# runtime stage
FROM alpine:latest

WORKDIR /loggerator
COPY --from=builder /app/loggerator .

CMD ["./loggerator", "-config", "/loggerator/config.yaml"]
