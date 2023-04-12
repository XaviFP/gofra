FROM ubuntu

COPY bin /bin
COPY config.yaml ./config.yaml

ENTRYPOINT [ "bin/gofra" ]