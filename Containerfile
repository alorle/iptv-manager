# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24.1

# First stage: the executable builder.
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY ./ ./

RUN go build -o /app .

# Final stage: the running container.
FROM scratch

ARG HTTP_ADDRESS=0.0.0.0
ENV HTTP_ADDRESS=${HTTP_ADDRESS}
ARG HTTP_PORT=8080
ENV HTTP_PORT=${HTTP_PORT}

COPY --from=builder /etc/passwd /etc/group /etc/
COPY --from=builder /app /app

USER nobody:nogroup

EXPOSE ${HTTP_PORT}

ENTRYPOINT ["/app"]
