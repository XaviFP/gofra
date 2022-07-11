/*
normie is a gofra plugin.
*/

package main

import (
	"github.com/XaviFP/gofra/gofra"
)

var Plugin plugin

type plugin struct{}

func (p plugin) Name() string {
	return "normie"
}

func (p plugin) Description() string {
	return "Just hanging 'round y'know?"
}

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	// Yeah, business as usual
}
