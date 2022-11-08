FROM golang:1.19.3-alpine3.16

WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
# Install Alpine Dependencies
RUN apk update && apk upgrade && apk add --update alpine-sdk && \
    apk add make cmake

RUN make all

ENTRYPOINT [ "bin/gofra" ]