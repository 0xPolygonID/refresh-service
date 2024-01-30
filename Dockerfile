FROM golang:1.19.7-bullseye as base

WORKDIR /build

COPY . .

RUN go mod download
RUN go build -o ./refresh-service .


FROM alpine:3.16.0

RUN apk add --no-cache libstdc++ gcompat libgomp

WORKDIR /app

COPY ./keys keys
COPY --from=base /build/refresh-service refresh-service
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV CIRCUITS_FOLDER_PATH=/app/keys

ENTRYPOINT ["/app/refresh-service"]
