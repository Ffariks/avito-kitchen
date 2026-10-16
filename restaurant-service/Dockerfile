FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/restaurant-service ./cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/restaurant-service /usr/local/bin/restaurant-service
EXPOSE 9090
ENTRYPOINT ["/usr/local/bin/restaurant-service"]
