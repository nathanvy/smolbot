# --- Builder stage --------------------------------------------------------
FROM golang:1.24-alpine AS builder

RUN addgroup -S app && adduser -S -G app app

WORKDIR /src

COPY go.mod Makefile ./

RUN go mod download

COPY . .

RUN apk add --no-cache make \
 && make release

# --- Final stage ----------------------------------------------------------
FROM alpine:3.18

RUN addgroup -S app && adduser -S -G app app

# for TLS (if needed)
RUN apk add --no-cache ca-certificates

WORKDIR /home/app

COPY --from=builder /src/smolbot-linux-amd64 ./smolbot

USER app

EXPOSE 21337

ENTRYPOINT ["./smolbot"]
