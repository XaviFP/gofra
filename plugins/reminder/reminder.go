/*
remind is a gofra plugin that allows users to set text-based reminders for themselves or other users
*/

package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"

	"gofra/gofra"
)

type plugin struct{}

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

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	g = gofra
	config = c
	g.Subscribe(
		"command/remind",
		p.Name(),
		handleReminder,
		0,
	)
}

func (p plugin) Run() {
	g.Logger.Info("Reminder Run method running . . .")
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
		err := g.SendStanza(r)
		if err != nil {
			g.Logger.Error(fmt.Sprintf("Error encoding message in Run() method of reminder Plugin: %v", err))
			continue
		}

		reminders, _ = pop(reminders)
	}
}

func handleReminder(e gofra.Event) gofra.Reply {
	argLine := e.Payload["commandBody"].(string)
	args := strings.Split(argLine, " ")
	/* !remind me tomorrow to buy milk
	 * !remind [target] [time] message:[message]
	 * !remind [message]
	 */
	if args[0] != config.Plugins["Commands"]["commandChar"].(string)+commandStr {
		if err := g.SendStanza(e.MB.Reply(config, "Wrong command")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{}
		}
	}

	//Remove command and leave just the args for it
	args = args[1:]

	if len(args) < 1 || (len(args) > 0 && args[0] == "") {
		if err := g.SendStanza(e.MB.Reply(config, "Need a message to remind")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{}
		}
	}

	msg, ok := e.GetStanza().(*gofra.MessageBody)
	if !ok {
		g.Logger.Debug(fmt.Sprintf("Ignoring packet: %T\n", e.GetStanza()))

		return gofra.Reply{Empty: true}
	}

	if msg == nil {
		g.Logger.Debug("Error msg is nil in command plugin")

		return gofra.Reply{Empty: true}
	}

	if msg.Body == "" {

		return gofra.Reply{Empty: true}
	}
	now := time.Now()
	segs := now.Unix()
	rmdr := reminder{time: segs + 10, to: msg.From, from: msg.From, msg: msg.Body, msgType: msg.Type}
	addReminder(rmdr)

	if err := g.SendStanza(e.MB.Reply(config, "Reminder added")); err != nil {
		g.Logger.Error(err.Error())
	}
	return gofra.Reply{Ok: true, Empty: false}
}

func addReminder(rmdr reminder) {
	reminders = append(reminders, rmdr)
	sort.Slice(reminders, func(i, j int) bool {

		return reminders[i].time < reminders[j].time
	})
}

func pop(reminders []reminder) ([]reminder, reminder) {

	return reminders[1:], reminders[0]
}

var Plugin plugin
