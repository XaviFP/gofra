/*
example is a gofra plugin that serves as a template to create new plugins.
*/

package main

import (
	"log"

	"gofra/gofra"
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
	if reply.Empty {
		r := &gofra.Reply{Empty: true}
		r.Ok = reply.Ok
		return r
	}

	// get reply's content and work with it
	data = reply.Payload
	log.Println(data)

	// return a reply
	return &gofra.Reply{Ok: true, Payload: data}
}
