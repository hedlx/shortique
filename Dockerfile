# syntax=docker/dockerfile:1

FROM golang:alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY *.go ./
RUN go mod download
RUN go build -o /shque


FROM alpine

RUN apk add yt-dlp
WORKDIR /
COPY --from=build /shque /shque

ENTRYPOINT ["/shque"]
