# syntax=docker/dockerfile:1

ARG NODE_VERSION=22.15.0
ARG GO_VERSION=1.24.1

# First stage: the frontend builder.
FROM node:${NODE_VERSION}-alpine AS frontend
WORKDIR /src

COPY ./package.json ./package-lock.json ./

RUN npm install

COPY ./ ./

RUN npm run build

# Second stage: the executable builder.
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY ./ ./
COPY --from=frontend /src/dist /src/dist

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
