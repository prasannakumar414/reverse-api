# syntax=docker/dockerfile:1

FROM golang:1.26.4-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/profile-retrieval-service \
    ./cmd

FROM alpine:3.23

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=build --chown=app:app \
    /out/profile-retrieval-service \
    /usr/local/bin/profile-retrieval-service

USER app

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["profile-retrieval-service"]
