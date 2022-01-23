# Gofra, an XMPP bot engine 
[https://ca.wikipedia.org/wiki/Gofra](Gofra) is a tiny XMPP bot engine written in Go.

As of now focuses on text-based commands.

Current design uses a golang plugin-based architechture as it was meant to be able to have it's plugins hot-reloaded (or even replaced or updated).  
Unfortunately, golang plugin system is far from mature and (at least in this case) adds more complexity than it solves. Plugins need to be compiled against the same version of the binary that is going to use them. Also, testing of binary plugin files cannot be performed. More info on https://github.com/golang/go/issues/27751  
As a matter of fact, Go 1.17 has a linker error crashing plugins accessing network resources.  

So, although it's been a good and fun learning experience your cents are better invested in either going monolithic or using tools like https://github.com/hashicorp/go-plugin instead.   
In that regard and due to the lack of support for plugin testing, Gofra will surely move away from the current plugin-based architecture.

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
For every MUC the bot needs to join, add an entry under `mucs:`.  
`mucJoinHistory` refers to the amount of previous messages in the muc the bot will ask the server for.

To add configuration options for your plugin, create an entry for your plugin under `plugins:`.    


## Building the project & running tests

Bulding and running a Docker image  

```
docker build -t gofra .
docker run --name gofra gofra
```

Building and running the binary  
```
make all && ./bin/gofra
```


To build the project and run the tests:

```
make tests
```

## Creating plugins

Gofra plugins must comply with the Plugin interface:
```
type Plugin interface {
  Name() string
  Description() string
  Init(Config, *Gofra)
}
```
As parameters of the Init method the plugin receives the API object which upon to perform calls, and also the configuration passed in to gofra.  

Aditionally, the Runnable interface can be implemented:
```
type Runnnable interface {
  Run()
}
```
The Run method is ran as a goroutine and is meant for plugins that require some code to be executed periodically.
As an example of this, the reminder plugin implements the Runnable interface to provide time-based reminders.
Other uses can be serving a webpage to display data gathered from Gofra or serving an API to manage Gofra through HTTP, for example.

An easy way to get a grasp is to copy the example plugin template and build from there.

## Events

Plugins subscribe to events and can trigger others.
The following list covers the current available events published by Gofra and its plugins:  

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
