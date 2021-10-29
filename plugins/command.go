package main

import (
	"fmt"
	"gofra/gofra"
	"os"
	"strings"

	"gosrc.io/xmpp/stanza"
)

type plugin string
const defaultCommandChar = "!"
var g gofra.API
var c gofra.Config

func (p plugin) Name() string {
	return "Commands"
}

func (p plugin) Description() string {
	return "Makes easy to create text-based plugin commands"
}

func (p plugin) Init(config gofra.Config, api gofra.API) {
	c = config
	checkConfig(c)
	g = api
	g.Subscribe(
		"messageReceived",
		p.Name(),
		handleMessage,
		gofra.Options{9999},
	)
}

func checkConfig(config gofra.Config) {
	commandChar, exists := config.Extra["commandChar"]
	if !exists || commandChar.(string) == "" {
		config.Extra["commandChar"] = defaultCommandChar
	}
}

func handleMessage(e gofra.Event, acc *gofra.Event) (gofra.Reply, gofra.Event) {
	msg, ok := e.Stanza.(stanza.Message)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", e.Stanza)
		return gofra.Reply{nil, false, true}, e
	}
	//_, _ = fmt.Fprintf(os.Stdout, "Body = %s - from = %s\n", msg.Body, msg.From)
	if msg.Body == "" {
		return gofra.Reply{nil, false, true}, e
	}
	command := ""
	if strings.HasPrefix(msg.Body, c.Extra["commandChar"].(string)) {
		command = strings.Split(msg.Body, " ")[0][1:]
	}
	msgType := stanza.MessageTypeChat
	to := msg.From
	if !msg.Attrs.Type.IsEmpty() && msg.Attrs.Type == stanza.MessageTypeGroupchat {
		msgType = stanza.MessageTypeGroupchat
		to = strings.Split(msg.From, "/")[0]
	}
	eventName := "command/" + command
	e.Payload["commandBody"] = msg.Body
	event := gofra.Event{eventName, e.Payload, e.Stanza}
	reply := g.Publish(event)
	
	if !reply.Empty && reply.Ok && reply.Reply != nil{
		r := stanza.Message{Attrs: stanza.Attrs{To: to, Type: msgType}, Body: reply.GetAnswer()}
		_ = g.SendStanza(r)
	}

	return reply, e
}

var Plugin plugin