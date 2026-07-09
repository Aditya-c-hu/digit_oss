# --- Build stage ---
FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w" -o /out/ws-calculator ./cmd/ws-calculator

# --- Runtime stage ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/ws-calculator /app/ws-calculator
USER app
EXPOSE 8091
ENTRYPOINT ["/app/ws-calculator"]
