FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -o refract ./cmd

FROM alpine:3.23

RUN sed -i 's/https/http/' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache ca-certificates && \
    apk add --no-cache iptables ipset iproute2

WORKDIR /app
COPY --from=builder /app/refract /app/refract

ENTRYPOINT ["/app/refract"]