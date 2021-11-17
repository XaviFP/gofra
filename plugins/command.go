/*
command is a gofra plugin that makes it easy to create text-based plugin commands
*/

package main

import (
	"fmt"
	"gofra/gofra"
	"log"
	"os"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

type plugin string

const defaultCommandChar = "!"
const name = "Commands"
var g *gofra.Gofra
var c gofra.Config

func (p plugin) Name() string {
	return "Commands"
}

func (p plugin) Description() string {
	return "Makes it easy to create text-based plugin commands"
}

func (p plugin) Init(config gofra.Config, api gofra.API) {
	c = config
	g, _ = api.(*gofra.Gofra)
	g.Subscribe(
		"messageReceived",
		p.Name(),
		handleMessage,
		gofra.Options{Priority: 9999},
	)
	checkConfig(c)
}

func checkConfig(config gofra.Config) {
	log.Print(config)
	pluginConfig, exists := config.Plugins[name]
	if !exists {
		config.Plugins[name] = map[string]interface{}{"commandChar": defaultCommandChar}
	}
	commandChar, exists := pluginConfig["commandChar"]
	if !exists || commandChar.(string) == "" {
		config.Plugins[name]["commandChar"] = defaultCommandChar
	}
}

func handleMessage(e gofra.Event, acc *gofra.Event) (gofra.Reply, gofra.Event) {
	msg, ok := e.GetStanza().(*gofra.MessageBody)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", e.GetStanza())
		return gofra.Reply{nil, false, true}, e
	}
	if msg == nil {
		g.Logger.Println("Error msg is nil in command plugin")
		return gofra.Reply{nil, false, true}, e
	}
	t, ok := e.GetTokenReadEncoder().(xmlstream.TokenReadEncoder)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "COULDN'T CAST TokenReadEncoder: %T\n", t)
		return gofra.Reply{nil, false, true}, e
	}
	if t == nil {
		g.Logger.Println("Error t is nil in command plugin")
		return gofra.Reply{nil, false, true}, e
	}

	if msg.Body == "" {
		return gofra.Reply{nil, false, true}, e
	}
	command := ""
	if !strings.HasPrefix(msg.Body, c.Plugins[name]["commandChar"].(string)) {
		return gofra.Reply{nil, false, true}, e
	}
	command = strings.Split(msg.Body, " ")[0][1:]
	msgType := msg.Type
	to := msg.From

	eventName := "command/" + command
	e.Payload["commandBody"] = msg.Body
	event := gofra.Event{eventName, e.Payload}
	reply := g.Publish(event)
	
	if !reply.Empty && reply.Ok && reply.Payload != nil{
		r := gofra.MessageBody{Message: stanza.Message{Type: msgType, To: to.Bare()}, Body: reply.GetAnswer()}
		_ = t.Encode(r)
	}

	return reply, e
}

var Plugin plugin