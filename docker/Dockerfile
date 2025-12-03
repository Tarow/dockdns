FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .
COPY internal/ internal/
COPY templates/ templates/
COPY static/ static/

RUN CGO_ENABLED=0 go build

FROM alpine:3.23

WORKDIR /app
COPY --from=builder /app/dockdns /usr/bin/dockdns

CMD ["dockdns"]

