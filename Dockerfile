# syntax=docker/dockerfile:1

FROM golang:alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o /shque


FROM alpine:3

RUN apk add --no-cache yt-dlp
WORKDIR /
COPY --from=build /shque /shque

ENTRYPOINT ["/shque"]

