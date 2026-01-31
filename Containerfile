# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.6

# BUILD STAGE (frontend)
FROM node:20-alpine AS frontend-builder

WORKDIR /src

COPY web-ui/package*.json ./
RUN npm ci

COPY web-ui/ ./
RUN npm run build

## BUILD STAGE (backend)
FROM golang:${GO_VERSION}-alpine AS backend-builder
WORKDIR /src

COPY ./go.* ./
RUN go mod download

COPY ./ ./
COPY --from=frontend-builder /src/dist ./ui/dist

RUN go build -o /app .

# FINAL STAGE
FROM scratch

ENV HTTP_ADDRESS=0.0.0.0
ENV HTTP_PORT=8080
ENV CACHE_DIR=/cache

VOLUME [ "/cache" ]

COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend-builder /etc/passwd /etc/group /etc/
COPY --from=backend-builder /app /app

USER nobody:nogroup

EXPOSE 8080

ENTRYPOINT ["/app"]
