FROM golang:1.22.5-bookworm as base

WORKDIR /build

COPY . .

RUN go mod download
RUN go build -o ./refresh-service .


FROM alpine:3.18.4

RUN apk add --no-cache libstdc++ gcompat libgomp; \
    apk add --update busybox>1.3.1-r0; \
    apk add --update openssl>3.1.4-r1

RUN apk add doas; \
    adduser -S dommyuser -D -G wheel; \
    echo 'permit nopass :wheel as root' >> /etc/doas.d/doas.conf;
RUN chmod g+rx,o+rx /

WORKDIR /app

COPY ./keys keys
COPY --from=base /build/refresh-service refresh-service

ENV CIRCUITS_FOLDER_PATH=/app/keys

ENTRYPOINT ["/app/refresh-service"]
