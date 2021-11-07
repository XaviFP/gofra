/*
naughty is a test gofra plugin that tries to crash gofra through panicking handler, Init() and Run() methods.
*/

package main

import (
	"gofra/gofra"
)

type plugin string

func (p plugin) Name() string {
	return "naughty"
}

func (p plugin) Description() string {
	return "Tries to crash gofra"
}

func (p plugin) Init(c gofra.Config, api gofra.API) {
	api.Subscribe(
		"naughtyCrash",
		p.Name(),
		naughtyCrash,
		gofra.Options{},
	)
	panic("naughtyInitCrash")
}

func (p plugin) Run() {
	panic("naughtyRunCrash")
}

func naughtyCrash(e gofra.Event, _ *gofra.Event) (gofra.Reply, gofra.Event){
	panic("naughtyCrash")
}

var Plugin plugin
