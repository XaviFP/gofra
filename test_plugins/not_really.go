/*
notReally is almost a test gofra plugin but not really.
*/

package main

import (
	"gofra/gofra"
)

type plugin string

func (p plugin) Name() string {
	return "notReally"
}

func (p plugin) Description() string {
	return "Not really a plugin"
}

// "My Init() signature is off for a plugin :S"
func (p plugin) Init() {
}

func notReally(e gofra.Event, _ *gofra.Event) (gofra.Reply, gofra.Event){
	return gofra.Reply{}, e
}

var Plugin plugin
