/*
example is a gofra plugin that serves as a template to create new plugins.
*/

package main

import (
	"log"

	"gofra/gofra"
)

type plugin string

var g gofra.API
var config gofra.Config

func (p plugin) Name() string {
	return "example"
}

func (p plugin) Description() string {
	return "Example plugin"
}

func (p plugin) Init(conf gofra.Config, api gofra.API) {
	g = api
	config = conf
	g.Subscribe(
		"exampleEvent",
		p.Name(),
		joinMUCs,
		gofra.Options{},
	)
}

func handleExampleEvent(e gofra.Event, _ *gofra.Event) (gofra.Reply, gofra.Event){
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
		r := gofra.Reply{Empty: true}
		r.Ok = reply.Ok
		return r, e
	}
	
	// get reply's content and work with it
	data = reply.Payload
	log.Println(data)

	// return a reply
	return gofra.Reply{Ok: true, Empty: false, Payload: data}, e
}

var Plugin plugin