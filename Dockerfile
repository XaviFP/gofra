FROM ubuntu

COPY gofra/bin /bin

ENTRYPOINT [ "bin/gofra" ]