# Gofra, an XMPP bot engine 
[Gofra](https://ca.wikipedia.org/wiki/Gofra) is a tiny XMPP bot engine written in Go.

As of now, the current focus is on text-based commands.

Current design uses a Go plugin-based architecture as it was meant to be able to have its plugins hot-reloaded (or even replaced or updated).  
Unfortunately, Go's plugin system is far from mature and (at least in this case) adds more complexity than it solves. Plugins need to be compiled against the same version of the binary that is going to use them. Also, testing of binary plugin files cannot be performed. More info on https://github.com/golang/go/issues/27751  
[As a matter of fact, Go 1.17 has a linker error crashing plugins accessing network resources.](https://github.com/zoncoen-sample/go1.17-linker-issue)  

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
As parameters of the Init method the plugin receives the API object which upon to perform calls, and also the configuration passed in to Gofra.  

Aditionally, the Runnable interface can be implemented:
```
type Runnnable interface {
  Run()
}
```
The Run method is ran as a goroutine and is meant for plugins that require some code to be executed periodically.
As an example of this, the reminder plugin implements the Runnable interface to provide time-based reminders.
Other uses can be serving a webpage to display data gathered from Gofra or serving an API to manage Gofra through HTTP, for example.

An easy way to get a grasp is to see how other plugins work and build from there.

## Events

Plugins subscribe to events and can trigger others.
The following list covers the current available events published by Gofra and its plugins:  

### Engine events list

- connected
- initialized
- messageReceived
- presenceReceived
- eventSubscribed

### Available plugin event list

- command/commandName
- muc/joinedRoom
- muc/getOccupants
- muc/occupantJoinedMuc
- muc/occupantLeftMuc
- muc/occupants


## Commands usage

### assetinfo
User: !assetinfo btc  
Gofra: Bitcoin is a peer-to-peer electronic cash system that allows participants to digitally transfer units of bitcoin without a trusted intermediary. Bitcoin combines a public transaction ledger (blockchain), a decentralized currency issuance algorithm (proof-of-work mining), and a transaction verification system (transaction script). Bitcoin has a supply cap of 21 million bitcoin, 95% of which will be mined by the year 2025. Bitcoin relies on Nakamoto consensus, or consensus implied by the longest blockchain that has accumulated the most computational effort. 

### price
User: !price btcusd   
Gofra: btcusd: 37567   

User: !price btceur  
Gofra: btceur: 33314.3  

### remind
User: !remind me call the mechanic in one second  
Gofra: Reminder added  
Gofra: User, call the mechanic   

### pick
User: !pick Tokyo, Osaka, Kyoto  
Gofra: Chose: Osaka  

User: !pick 2 Strawberry, Chocolate, Vanilla, Caramel  
Gofra: Chose: Caramel and Vanilla  

### dice
User: !dice   
Gofra: 1d6: 6

User: !dice 3d20  
Gofra: 3d20: 17, 6, 16

### trivia
User: !trivia categories   
Gofra:   
9: General Knowledge  
16: Entertainment: Board Games  
17: Science & Nature  
18: Science: Computers  
.  
.  
.  
  
  
User: !trivia start 18   
Gofra:  
Science: Computers  
easy  
RAM stands for Random Access Memory.  
A) False  
B) True   

User: B  

Gofra:  
Science: Computers  
medium
Which coding language was the #1 programming language in terms of usage on GitHub in 2015?  
A) PHP  
B) JavaScript  
C) C#  
D) Python    
.  
.  
.  


### st (session tracker)
User: !st start  
Gofra: Session started!  

User: !st add reviewing code  
Gofra: Task added  

User: !st add very important meeting  
Gofra: Task added  

User: !st stop  
Gofra: Session status: Stopped  

Started at: 2022-Jan-26 08:28:17 AM  
Duration: 1m18s  
Tasks during session:  
1- reviewing code. Started at: 2022-Jan-26 08:28:49 AM  
2- very important meeting. Started at: 2022-Jan-26 08:29:21 AM  

