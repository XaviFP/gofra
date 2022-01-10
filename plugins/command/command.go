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
)

type plugin struct{}

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

func (p plugin) Init(config gofra.Config, gofra *gofra.Gofra) {
	c = config
	g = gofra
	g.Subscribe(
		"messageReceived",
		p.Name(),
		handleMessage,
		9999,
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

func handleMessage(e gofra.Event) gofra.Reply {
	msg, ok := e.GetStanza().(gofra.MessageBody)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", e.GetStanza())
		return gofra.Reply{Empty: true}
	}

	if msg.Body == "" {
		return gofra.Reply{Empty: true}
	}

	command := ""
	if !strings.HasPrefix(msg.Body, c.Plugins[name]["commandChar"].(string)) {
		return gofra.Reply{Empty: true}
	}

	command = strings.Split(msg.Body, " ")[0][1:]
	// msgType := msg.Type
	// to := msg.From

	eventName := "command/" + command

	event := gofra.Event{Name: eventName, MB: msg, Payload: e.Payload}
	reply := g.Publish(event)

	// if !reply.Empty && reply.GetAnswer() != "" {
	// 	r := gofra.MessageBody{Message: stanza.Message{Type: msgType, To: to.Bare()}, Body: reply.GetAnswer()}
	// 	err := g.Client.Encode(g.Context, r)

	// 	if err != nil {
	// 		g.Logger.Println("Error encoding message in command Plugin: ", err)
	// 	}
	// }
	return reply
}

var Plugin plugin
