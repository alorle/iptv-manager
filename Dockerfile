# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.7
ARG NODE_VERSION=22

# BUILD STAGE (frontend)
FROM node:${NODE_VERSION}-alpine AS frontend-builder

WORKDIR /src

COPY package.json package-lock.json ./
COPY ui/package.json ui/
RUN npm ci

COPY ui/ ui/
RUN npm run build -w ui

## BUILD STAGE (backend)
FROM golang:${GO_VERSION}-alpine AS backend-builder
WORKDIR /src

COPY ./go.* ./
RUN go mod download

COPY ./ ./
COPY --from=frontend-builder /src/ui/dist ./ui/dist

RUN go build -o /app ./cmd/iptv-manager

# FINAL STAGE
FROM scratch

EXPOSE 8080
ENV PORT=8080

ENV DB_PATH=/data/database.db
VOLUME [ "/data" ]

COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend-builder /etc/passwd /etc/group /etc/
COPY --from=backend-builder /app /app

USER nobody:nogroup

ENTRYPOINT ["/app"]
