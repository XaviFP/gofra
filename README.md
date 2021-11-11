# gofra, an XMPP bot engine 

## Config
--------

TBD  

Fields of the config file and convention for per-plugin configuration  


## Building & running tests
--------------------------

To build:  

```
docker build -t gofra .
docker run --name gofra gofra
```

For tests:  

```
make tests
go test -p=1 -coverprofile=coverage.out  && go tool cover -html=coverage.out
```

## Creating plugins
-----------------------

Easiest way is to copy the example plugin template and build from there.


## Engine events list
--------------------

- Connected
- Initialized
- MessageReceived
- PresenceReceived
- IQReceived
- EventSubscribed

## Available plugin event list
-----------------------------

- Command/commandName
- JoinedMuc
- LeftMuc
