FROM ubuntu

COPY bin /bin
ADD config.yaml ./config.yaml

ENTRYPOINT [ "bin/gofra" ]