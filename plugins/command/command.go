/*
command is a gofra plugin that makes it easy to create text-based plugin commands
*/

package main

import (
	"strings"

	"gofra/gofra"
)

var Plugin plugin

var commandChar = "!"

var g *gofra.Gofra
var c gofra.Config

type plugin struct{}

func (p plugin) Name() string {
	return "Commands"
}

func (p plugin) Description() string {
	return "Makes it easy to create text-based plugin commands"
}

func (p plugin) Init(config gofra.Config, gofra *gofra.Gofra) {
	c = config
	g = gofra

	p.checkConfig(c)

	g.Subscribe(
		"messageReceived",
		p.Name(),
		handleMessage,
		1,
	)
}

func (p plugin) checkConfig(config gofra.Config) {
	pluginConfig, exists := config.Plugins[p.Name()]
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
	if e.MB.Body == "" {
		return nil
	}

	command := ""
	if !strings.HasPrefix(e.MB.Body, commandChar) {
		return nil
	}

	command = strings.Fields(e.MB.Body)[0][1:]
	eventName := "command/" + command

	event := gofra.Event{
		Name:    eventName,
		MB:      e.MB,
		Payload: e.Payload,
	}

	return g.Publish(event)
}
