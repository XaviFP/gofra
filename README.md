# gofra, an XMPP bot engine 

## Config
---------

TBD  

Fields of the config file and convention for per-plugin configuration  


## Building the project & running tests
---------------------------------------

To build a docker image and run a container out of it you can do as follows:

```
docker build -t gofra .
docker run --name gofra gofra
```
In case you only want to build the binary and run it, you can do like so:
```
make all && ./bin/gofra
```


To build the project and run the tests:

```
make tests
go test -p=1 -coverprofile=coverage.out  && go tool cover -html=coverage.out
```

## Creating plugins
-----------------------

Gofra plugins must comply with the Plugin interface:
```
Name()
Description()
Init(gofra.Config, gofra.API)
```
As parameters of the Init method the plugin receives the API object which upon to perform calls, and also the configuration passed in to gofra.
Aditionally, the Runnable interface can be implemented:
```
Run()
```
The Run method is meant for plugins that need an endless loop like an HTTP server for instance.
It is ran as a goroutine.

An easy way to get a grasp is to copy the example plugin template and build from there.

## Events
---------

Plugins subscribe to events and can trigger others, here's a list of them:

### Engine events list

- Connected
- Initialized
- MessageReceived
- PresenceReceived
- IQReceived
- EventSubscribed

### Available plugin event list

- Command/commandName
- JoinedMuc
- LeftMuc
