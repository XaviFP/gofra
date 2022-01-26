FROM golang:1.16.10-alpine3.14

WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
# Install Alpine Dependencies
RUN apk update && apk upgrade && apk add --update alpine-sdk && \
    apk add make cmake

RUN make all

ENTRYPOINT [ "bin/gofra" ]