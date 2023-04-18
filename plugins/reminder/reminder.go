/*
remind is a gofra plugin that allows users to set text-based reminders for themselves or other users
*/

package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"

	"github.com/XaviFP/gofra/internal"
)

var Plugin plugin

var g *gofra.Gofra
var reminders []reminder
var dueReminders = make(chan reminder, 10)
var newReminders = make(chan reminder, 10)
var sendReminders = make(chan reminder, 10)
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

func (p plugin) Help() string {
	reply := g.Publish(gofra.Event{Name: "command/getCommandChar", MB: gofra.MessageBody{}, Payload: nil})
	commandChar := reply.GetAnswer()
	return fmt.Sprintf("Usage: Format %sremind [nick] [text to remind] [time to remind]\n[nick] can be omitted on 1 to 1 converstation with the bot. \"me\" can be used in a MUC setting if reminder is for oneself. \n Example: \n%sremind me call the mechanic in one second -> Reminder added", commandChar, commandChar)
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
	go reminderMonitor()
	go loadState()
}

func (p plugin) Run() {
	for rmdr := range sendReminders {

		r := gofra.MessageBody{Message: stanza.Message{Type: rmdr.msgType, To: rmdr.to.Bare()}, Body: rmdr.msg}

		err := g.SendStanza(r)
		if err != nil {
			g.Logger.Error(fmt.Sprintf("Error encoding message in Run() method of reminder Plugin: %v", err))
		}
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

	newReminders <- rmdr

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
	g.Logger.Info(fmt.Sprintf("REMINDER ADDED %v", rmdr))
}

func pop(rmdr reminder) {
	var index int
	for i, reminder := range reminders {
		if reminder.time == rmdr.time &&
			reminder.msg == rmdr.msg &&
			reminder.msgType == rmdr.msgType {
			index = i
		}
	}
	g.Logger.Info(fmt.Sprintf("REMINDER REMOVED %v", reminders[index]))
	if index == 0 {
		reminders = reminders[1:]
	} else if index == len(reminders)-1 {
		reminders = reminders[:len(reminders)-1]
	} else {
		reminders = append(reminders[:index], reminders[index+1:]...)
	}
}

func persistState() {
	g.Logger.Info("INTO PERSIST STATE")
	var state strings.Builder
	for _, reminder := range reminders {
		state.WriteString(fmt.Sprintf("%d %s %s %s %s\n", reminder.time, reminder.msgType, reminder.from, reminder.to, reminder.msg))
	}

	file, err := os.Create("/data/reminders.txt")
	if err != nil {
		g.Logger.Error(err.Error())
		return
	}
	g.Logger.Info("OPENED FILE SUCCESSFULLY")
	defer file.Close()

	_, err = file.WriteString(state.String())
	if err != nil {
		g.Logger.Error(err.Error())
	}
	g.Logger.Info("GOING OUT OF PERSIST STATE")
}

func loadState() {
	file, err := os.Open("/data/reminders.txt")
	if err != nil {
		g.Logger.Error(err.Error())
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		args := strings.Fields(scanner.Text())
		if len(args) < 5 {
			continue
		}

		time, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			continue
		}

		msgType := stanza.MessageType(args[1])

		from, err := jid.Parse(args[2])
		if err != nil {
			continue
		}

		to, err := jid.Parse(args[3])
		if err != nil {
			continue
		}

		msg := strings.Join(args[4:], " ")

		rmdr := reminder{
			time,
			to,
			from,
			msg,
			msgType,
		}
		g.Logger.Info(fmt.Sprintf("REMINDER LOADED FROM FILESYSTEM %v", rmdr))
		addReminder(rmdr)
		go waitTimer(rmdr)
	}
	// Handle reason of stop
	if err := scanner.Err(); err != nil {
		g.Logger.Error("Broken file stream " + err.Error())
		return
	}
	// If error is nil means error was EOF
}

func reminderMonitor() {
	for {
		select {
		case rmdr := <-newReminders:
			addReminder(rmdr)
			go waitTimer(rmdr)
			persistState()
		case rmdr := <-dueReminders:
			pop(rmdr)
			persistState()
			sendReminders <- rmdr
		}
	}
}

func waitTimer(rmdr reminder) {
	var tmr *time.Timer
	if rmdr.time <= time.Now().Unix() {
		tmr = time.NewTimer(time.Unix(time.Now().Unix()+1, 0).Sub(time.Now()))
	} else {
		tmr = time.NewTimer(time.Unix(rmdr.time, 0).Sub(time.Now()))
	}
	<-tmr.C
	dueReminders <- rmdr
}
