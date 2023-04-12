/*
dice is a gofra plugin that provides a utility to simulate dice throws
*/

package main

import (
	"fmt"
	"strings"

	"github.com/XaviFP/gofra/internal"
)

var Plugin plugin

var g *gofra.Gofra
var config gofra.Config
var defaultDiceFaces = 6
var defaultDiceQuantity = 1

type throw struct {
	quantity int
	faces    int
}

type plugin struct{}

func (p plugin) Name() string {
	return "Help"
}

func (p plugin) Description() string {
	return "Provides help with instructions on how to use other plugins"
}

func (p plugin) Help() string {
	reply := g.Publish(gofra.Event{Name: "command/getCommandChar", MB: gofra.MessageBody{}, Payload: nil})
	commandChar := reply.GetAnswer()
	return fmt.Sprintf("Usage: %shelp [plugin]\nFor a list of plugins invoke without arguments", commandChar)
}

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	g = gofra
	config = c

	g.Subscribe(
		"command/help",
		p.Name(),
		handleCommand,
		0,
	)
}

func handleCommand(e gofra.Event) *gofra.Reply {
	args := strings.Fields(e.MB.Body)[1:]
	var answer strings.Builder
	plugins := g.GetPlugins()
	// invoked without args provides list of plugins and their description
	if len(args) == 0 {
		for name, plugin := range plugins {
			answer.WriteString(fmt.Sprintf("%s: %s\n", name, plugin.Description()))
		}
		if err := g.SendStanza(e.MB.Reply(answer.String())); err != nil {
			g.Logger.Error(err.Error())

			return nil
		}
	}

	for _, arg := range args {
		p, exists := plugins[arg]
		if !exists {
			answer.WriteString(fmt.Sprintf("Plugin %s not found\n", arg))
			continue
		}

		answer.WriteString(fmt.Sprintf("%s: %s\n", arg, p.Help()))
	}

	if err := g.SendStanza(e.MB.Reply(answer.String())); err != nil {
		g.Logger.Error(err.Error())

		return nil
	}

	return nil
}
