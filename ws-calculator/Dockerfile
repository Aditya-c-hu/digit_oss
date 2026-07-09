FROM golang:1.26.3-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN apk add --no-cache git && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /app/ws-calculator ./cmd/ws-calculator

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/ws-calculator /usr/local/bin/ws-calculator
EXPOSE 8091
ENTRYPOINT ["/usr/local/bin/ws-calculator"]
