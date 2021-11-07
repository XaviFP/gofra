/*
normie is a gofra plugin.
*/

package main

import (
	"gofra/gofra"
)

type plugin string

func (p plugin) Name() string {
	return "normie"
}

func (p plugin) Description() string {
	return "Just hanging 'round y'know?"
}

func (p plugin) Init(c gofra.Config, api gofra.API) {
	// Yeah, business as usual
}

var Plugin plugin
