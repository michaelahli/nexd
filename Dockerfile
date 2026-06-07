# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nextd ./cmd/nextd

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/nextd /usr/local/bin/nextd

EXPOSE 8080

ENTRYPOINT ["nextd"]
