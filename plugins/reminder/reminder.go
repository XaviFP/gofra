/*
remind is a gofra plugin that allows users to set text-based reminders for themselves or other users
*/

package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"

	"github.com/XaviFP/gofra/gofra"
)

var Plugin plugin

var g *gofra.Gofra
var reminders []reminder
var occupants = make(map[string][]string)
var w = when.New(nil)

type reminder struct {
	time    int64
	to      jid.JID
	from    jid.JID
	msg     string
	msgType stanza.MessageType
}

type plugin struct{}

func (p plugin) Name() string {
	return "Remind"
}

func (p plugin) Description() string {
	return "Reminds you or another recipient of something you noted"
}

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	g = gofra
	g.Subscribe(
		"command/remind",
		p.Name(),
		handleReminder,
		0,
	)
	g.Subscribe(
		"muc/occupants",
		p.Name(),
		handleOccupants,
		0,
	)

	w.Add(en.All...)
	w.Add(common.All...)
}

func (p plugin) Run() {
	for {
		time.Sleep(1 * time.Second)

		if len(reminders) < 1 {
			continue
		}

		rmdr := reminders[0]
		if rmdr.time > time.Now().Unix() {
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

func handleReminder(e gofra.Event) *gofra.Reply {
	msg := e.MB
	args := strings.Fields(msg.Body)[1:]

	if len(args) < 1 || (len(args) > 0 && args[0] == "") {
		if err := g.SendStanza(e.MB.Reply("Need a message to remind")); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil
	}

	t, err := w.Parse(msg.Body, time.Now())
	if err != nil {
		g.Logger.Error(err.Error())
		if err := g.SendStanza(e.MB.Reply("Couldn't parse date")); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil
	}

	answer := ""
	if msg.Type == stanza.GroupChatMessage {
		_, isParticipant := isOccupant(msg.From.Bare().String(), args[0])

		if args[0] == "me" || !isParticipant {
			answer += msg.From.Resourcepart() + ", "
		} else {
			answer += args[0] + ", "
		}

		answer += strings.Join(args[1:], " ")

	} else {
		answer += strings.Join(args, " ")
	}

	answer = strings.Replace(answer, t.Text, "", -1)
	rmdr := reminder{
		time:    t.Time.Unix(),
		to:      msg.From,
		from:    msg.From,
		msg:     answer,
		msgType: msg.Type,
	}
	addReminder(rmdr)

	if err := g.SendStanza(e.MB.Reply("Reminder added")); err != nil {
		g.Logger.Error(err.Error())
	}

	return nil
}

func handleOccupants(e gofra.Event) *gofra.Reply {
	occupants = e.Payload["occupants"].(map[string][]string)

	return nil
}

func isOccupant(room, occupant string) (int, bool) {
	position := -1
	for index, occ := range occupants[room] {
		if occ == occupant {
			position = index
			break
		}
	}

	return position, position != -1
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
