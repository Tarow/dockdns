FROM golang:1.21.4-alpine as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build

FROM alpine:3.18

RUN apk add curl

WORKDIR /app
COPY --from=builder /app/dockdns /usr/bin/dockdns

CMD ["dockdns"]

