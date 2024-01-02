FROM golang:1.21.4-alpine as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .
COPY internal/ internal/

RUN CGO_ENABLED=0 go build

FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/dockdns /usr/bin/dockdns

CMD ["dockdns"]

