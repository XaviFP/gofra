/*
remind is a gofra plugin that allows users to set text-based reminders for themselves or other users
*/

package main

import (
	//"log"
	"sort"
	"strings"
	"time"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"

	"gofra/gofra"
)

type plugin string

type reminder struct {
	time    int64
	to      jid.JID
	from    jid.JID
	msg     string
	msgType stanza.MessageType
}

const commandStr = "remind"

var g *gofra.Gofra
var config gofra.Config
var reminders []reminder

func (p plugin) Name() string {
	return "Remind"
}

func (p plugin) Description() string {
	return "Reminds you or another recipient of something you noted"
}

func (p plugin) Init(c gofra.Config, api gofra.API) {
	g, _ = api.(*gofra.Gofra)
	config = c
	g.Subscribe(
		"command/remind",
		p.Name(),
		handleReminder,
		gofra.Options{},
	)
}

func (p plugin) Run() {
	g.Logger.Println("Reminder Run method running . . .")
	for {
		time.Sleep(1 * time.Second) // wait 1 sec
		now := time.Now()
		segs := now.Unix()
		if len(reminders) < 1 {
			continue
		}
		rmdr := reminders[0]
		if rmdr.time > segs {
			continue
		}

		r := gofra.MessageBody{Message: stanza.Message{Type: rmdr.msgType, To: rmdr.to.Bare()}, Body: rmdr.msg}
		trc, err := g.Client.EncodeMessage(g.Context, r)
		trc.Close()
		if err != nil {
			g.Logger.Println("Error encoding message in Run() method of reminder Plugin: ", err)
			continue
		}
		reminders, _ = pop(reminders)
	}
}

func handleReminder(e gofra.Event) gofra.Reply {
	var r gofra.Reply
	argLine := e.Payload["commandBody"].(string)
	args := strings.Split(argLine, " ")
	/* !remind me tomorrow to buy milk
	 * !remind [target] [time] message:[message]
	 * !remind [message]
	 */
	if args[0] != config.Plugins["Commands"]["commandChar"].(string)+commandStr {
		r = gofra.Reply{Ok: false, Empty: false}
		r.SetAnswer("Wrong command")
		return r
	}

	//Remove command and leave just the args for it
	args = args[1:]

	if len(args) < 1 || (len(args) > 0 && args[0] == "") {
		r = gofra.Reply{Ok: false, Empty: false}
		r.SetAnswer("Need a message to remind")
		return r
	}

	msg, ok := e.GetStanza().(*gofra.MessageBody)
	if !ok {
		g.Logger.Printf("Ignoring packet: %T\n", e.GetStanza())
		return gofra.Reply{Empty: true}
	}
	if msg == nil {
		g.Logger.Println("Error msg is nil in command plugin")
		return gofra.Reply{Empty: true}
	}

	if msg.Body == "" {
		return gofra.Reply{Empty: true}
	}
	now := time.Now()
	segs := now.Unix()
	rmdr := reminder{time: segs + 10, to: msg.From, from: msg.From, msg: msg.Body, msgType: msg.Type}
	reminders = append(reminders, rmdr)
	r = gofra.Reply{Ok: true, Empty: false}
	r.SetAnswer("Reminder added")
	return r
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
