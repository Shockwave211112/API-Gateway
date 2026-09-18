FROM golang:1.27.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /gateway ./

# ---- run stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /gateway .
COPY .env .env

EXPOSE 8080

ENTRYPOINT ["./gateway"]