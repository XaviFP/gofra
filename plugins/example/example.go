/*
example is a gofra plugin that serves as a template to create new plugins.
*/

package main

import (
	"fmt"
	"log"

	"github.com/XaviFP/gofra/internal"
)

var Plugin plugin

type plugin struct{}

var g *gofra.Gofra
var config gofra.Config

func (p plugin) Name() string {
	return "example"
}

func (p plugin) Description() string {
	return "Example plugin"
}

func (p plugin) Help() string {
	// if the plugin is a command the following provides information on how to use it
	reply := g.Publish(gofra.Event{Name: "command/getCommandChar", MB: gofra.MessageBody{}, Payload: nil})
	commandChar := reply.GetAnswer()
	return fmt.Sprintf("Usage: %sexampleplugin first_argument second_argument ...", commandChar)
}

func (p plugin) Init(conf gofra.Config, api *gofra.Gofra) {
	g = api
	config = conf

	g.Subscribe(
		"exampleEvent",
		p.Name(),
		handleExampleEvent,
		0,
	)
}

func handleExampleEvent(e gofra.Event) *gofra.Reply {
	// do things with e
	data := e.Payload
	log.Println(data)

	// maybe trigger another event
	reply := g.Publish(
		gofra.Event{
			Name: "newExampleEvent",
		})
	// if reply is empty return
	if reply == nil {
		return reply
	}

	// get reply's content and work with it
	data = reply.Payload
	g.Logger.Info(fmt.Sprintf("%v", data))

	// return a reply
	return &gofra.Reply{Payload: data}
}
