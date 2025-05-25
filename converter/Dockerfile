FROM golang:1.24.3-alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 0

ENV GOOS linux

RUN apk update --no-cache && apk add --no-cache tzdata

WORKDIR /build

ADD go.mod .

ADD go.sum .

RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o /app/converter .

FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates ffmpeg

COPY --from=builder /usr/share/zoneinfo/Europe/London /usr/share/zoneinfo/Europe/London

ENV TZ Europe/London

WORKDIR /app

COPY --from=builder /app/converter /app/converter

CMD ["./converter"]