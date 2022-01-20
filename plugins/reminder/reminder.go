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
	//"github.com/olebedev/when/rules"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"

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
var occupants = make(map[string][]string)
var w = when.New(nil)

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
	w.Add(en.All...)
	w.Add(common.All...)
}

func (p plugin) Run() {
	g.Logger.Info("Reminder Run method running . . .")
	fiveSecs := 0
	for {
		time.Sleep(1 * time.Second) // wait 1 sec
		fiveSecs++
		if fiveSecs % 5 == 0 {
			r := g.Publish(gofra.Event{Name: "muc/getOccupants"})
			o, ok := r.Payload["occupants"].(map[string][]string)
			if ok {
				occupants = o
			}
			fiveSecs = 0
		}

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

func handleReminder(e gofra.Event) *gofra.Reply {
	argLine := e.MB.Body
	args := strings.Split(argLine, " ")
	/* !remind me tomorrow to buy milk
	 * !remind [target] [time] message:[message]
	 * !remind [message]
	 */
	if args[0] != config.Plugins["Commands"]["commandChar"].(string)+commandStr {
		if err := g.SendStanza(e.MB.Reply("Wrong command")); err != nil {
			g.Logger.Error(err.Error())

			return nil
		}
	}

	//Remove command and leave just the args for it
	args = args[1:]

	if len(args) < 1 || (len(args) > 0 && args[0] == "") {
		if err := g.SendStanza(e.MB.Reply("Need a message to remind")); err != nil {
			g.Logger.Error(err.Error())

			return nil
		}
	}

	msg := e.MB

	if msg.Body == "" {

		return nil
	}

	t, err := w.Parse(msg.Body, time.Now())
	if err != nil {
		g.Logger.Debug(fmt.Sprintf("%v", err))
		if err := g.SendStanza(e.MB.Reply("Couldn't parse date")); err != nil {
			g.Logger.Error(err.Error())
			return nil
		}
	}
	g.Logger.Debug(fmt.Sprintf("%v", t))

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
	rmdr := reminder{time: t.Time.Unix(), to: msg.From, from: msg.From, msg: answer, msgType: msg.Type}
	addReminder(rmdr)

	if err := g.SendStanza(e.MB.Reply("Reminder added")); err != nil {
		g.Logger.Error(err.Error())
	}
	return &gofra.Reply{Ok: true}
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

var Plugin plugin
