/*
command is a gofra plugin that makes it easy to create text-based plugin commands
*/

package main

import (
	"fmt"
	"os"
	"strings"

	"gofra/gofra"
)

var Plugin plugin

type plugin struct{}

var commandChar = "!"
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
		1,
	)
	checkConfig(c)
}

func checkConfig(config gofra.Config) {
	pluginConfig, exists := config.Plugins[name]
	if !exists {
		g.Logger.Warn("No config for plugin Commands")

		return
	}

	char, exists := pluginConfig["commandChar"]
	cChar, ok := char.(string)
	if !exists || !ok || cChar == "" {
		g.Logger.Warn("No config for plugin Commands")

		return
	}

	commandChar = cChar
}

func handleMessage(e gofra.Event) *gofra.Reply {
	msg, ok := e.GetStanza().(gofra.MessageBody)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", e.GetStanza())
		return nil
	}

	if msg.Body == "" {
		return nil
	}

	command := ""
	if !strings.HasPrefix(msg.Body, commandChar) {
		return nil
	}

	command = strings.Split(msg.Body, " ")[0][1:]
	eventName := "command/" + command

	event := gofra.Event{Name: eventName, MB: msg, Payload: e.Payload}
	reply := g.Publish(event)

	return reply
}
