# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.6

## BUILD STAGE
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

COPY ./go.* ./

RUN go mod download

COPY ./ ./

RUN go build -o /app .

# FINAL STAGE
FROM scratch

ENV HTTP_ADDRESS=0.0.0.0
ENV HTTP_PORT=8080
ENV CACHE_DIR=/cache

VOLUME [ "/cache" ]

COPY --from=builder /etc/passwd /etc/group /etc/
COPY --from=builder /app /app

USER nobody:nogroup

EXPOSE 8080

ENTRYPOINT ["/app"]
