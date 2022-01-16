# Gofra, an XMPP bot engine 
Gofra is a tiny XMPP bot engine written in Go.

As of now focuses on text-based commands.

Current design uses a golang plugin-based architechture as it was meant to be able to have it's plugins hot-reloaded (or even replaced or updated).  
Unfortunately, golang plugin system is far from mature and (at least in this case) adds more complexity than it solves. Plugins need to be compiled against the same version of the binary that is going to use them. Also, testing of binary plugin files cannot be performed. More info on https://github.com/golang/go/issues/27751  
As a matter of fact, Go 1.17 has a linker error crashing plugins accessing network resources.  

So, although it's been a good and fun learning experience your cents are better invested in either going monolithic or using tools like https://github.com/hashicorp/go-plugin instead.

## Config
Config fields look as follows:

```
jid: "account@server.tld"
password: "m0r3,S3cur3,Th4n,1234."
nick: "Gofra"
debug: true
logXML: true
pluginPaths:
  - "bin/plugins/"

mucs:
  - mucNick: "Gofra"
    mucJoinHistory: 0
    mucJid: "mucJid@mucService.server.tld"
    mucPassword: "open,sesame"

plugins:
  Commands:
    commandChar: "!"
  Dice:
    defaultDice: 6
```
For every MUC the bot needs to join, add an entry under `mucs:`. `mucJoinHistory` refers to the amount of previous messages in the muc the bot will ask the server for.

To add configuration options for your plugin, create an entry for your plugin under `plugins:` and add them.


## Building the project & running tests

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

Gofra plugins must comply with the Plugin interface:
```
Name() string
Description() string
Init(Config, *Gofra)
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

Plugins subscribe to events and can trigger others, here's a list of them:

### Engine events list

- Connected
- Initialized
- MessageReceived
- PresenceReceived
- EventSubscribed

### Available plugin event list

- Command/commandName
- JoinedMuc
- LeftMuc
