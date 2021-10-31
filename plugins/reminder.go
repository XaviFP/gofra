/*
remind is a gofra plugin that allows users to set text-based reminders for themselves or other users
*/

package main

import (
	"fmt"
	"gofra/gofra"
	"sort"
	"strings"
	"time"

	"gosrc.io/xmpp/stanza"
)

type plugin string

type reminder struct {
	time    int64
	to      string
	from    string
	msg     string
	msgType stanza.StanzaType
}

const commandStr = "remind"

var g gofra.API
var config gofra.Config
var reminders []reminder

func (p plugin) Name() string {
	return "Remind"
}

func (p plugin) Description() string {
	return "Reminds you or another recipient of something you noted"
}

func (p plugin) Init(c gofra.Config, api gofra.API) {
	g = api
	config = c
	g.Subscribe(
		"command/remind",
		p.Name(),
		handleReminder,
		gofra.Options{},
	)
	g.Subscribe(
		"connected",
		p.Name(),
		sayHello,
		gofra.Options{Priority: 0},
	)
	g.Subscribe(
		"occupantJoinedMuc",
		p.Name(),
		joined,
		gofra.Options{Priority: 0},
	)
	g.Subscribe(
		"mucJoined",
		p.Name(),
		joined,
		gofra.Options{Priority: 0},
	)
}


func joined(e gofra.Event, _ *gofra.Event) (gofra.Reply, gofra.Event){
	if e.Name == "mucJoined"{
		fmt.Println("I JOINED A ROOM")
		err := g.Send("vaulor@blastersklan.com", "I JOINED A ROOM", stanza.MessageTypeChat)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		fmt.Println("SOMEONE JOINED A ROOM")
		err := g.Send("vaulor@blastersklan.com", "SOMEONE JOINED A ROOM", stanza.MessageTypeChat)
		if err != nil {
			fmt.Println(err)
		}
	}
	return gofra.Reply{Empty: true}, e
}


func sayHello(e gofra.Event, _ *gofra.Event) (gofra.Reply, gofra.Event){
	// shigoto@agora.blastersklan.com/Vaulor !remind klisahfdliasufgdh chat
	err := g.Send("vaulor@blastersklan.com", "Initialized and sending", stanza.MessageTypeChat)
	if err != nil {
		fmt.Println(err)
	}
	return gofra.Reply{Empty: true}, e
}

func (p plugin) Run() {
	fmt.Println("Running . . .")
	for {
		time.Sleep(1 * time.Second) // wait 1 sec
		now := time.Now()
		segs := now.Unix()
		/* fmt.Println(segs) */
		if len(reminders) < 1 {
			continue
		}
		rmdr := reminders[0]
		fmt.Println(rmdr.time, segs)
		if rmdr.time > segs {
			continue
		}
		err := g.Send(rmdr.to, rmdr.msg, stanza.StanzaType(rmdr.msgType))
		if err != nil {
		}
		reminders, _ = pop(reminders)
	}
}

func handleReminder(e gofra.Event, _ *gofra.Event) (gofra.Reply, gofra.Event){
	var r gofra.Reply
	argLine := e.Payload["commandBody"].(string)
	args := strings.Split(argLine, " ")
	/* !remind me tomorrow to buy milk
	 * !remind [target] [time] message:[message]
	 * !remind [message]
	 */
	 if args[0] != config.Extra["commandChar"].(string) + commandStr {
		r = gofra.Reply{Ok: false, Empty: false}
		r.SetAnswer("Wrong command")
		return r, e 
	 }

	//Remove command and leave just the args for it
	args = args[1:]
	if len(args) < 1 {
		r = gofra.Reply{Ok: false, Empty: false}
		r.SetAnswer("Need a message to remind")
		return r, e
	}
	msg, ok := e.Stanza.(stanza.Message)
	if !ok {
	}
	now := time.Now()
	segs := now.Unix()
	rmdr := reminder{time: segs + 10, to: msg.From, from: msg.From, msg: msg.Body, msgType: stanza.MessageTypeChat}
	reminders = append(reminders, rmdr)
	fmt.Println(reminders)
	r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("Reminder added")
		return r, e
}

func addReminder(reminders []reminder, rmdr reminder) []reminder {
	reminders = append(reminders, rmdr)
	sort.Slice(reminders, func(i, j int) bool {
		return reminders[i].time < reminders[j].time
	})
	return reminders
}

func pop(reminders []reminder) ([]reminder, reminder) {
	return reminders[1:], reminders[0]
}

var Plugin plugin
