FROM ubuntu

COPY bin /bin
COPY config.yaml ./config.yaml

VOLUME [ "/data" ]
ENTRYPOINT [ "bin/gofra" ]
