/*
notReally is almost a test gofra plugin but not really.
*/

package main

var Plugin plugin

type plugin struct{}

func (p plugin) Name() string {
	return "notReally"
}

func (p plugin) Description() string {
	return "Not really a plugin"
}

// "My Init() signature is off for a plugin :S"
func (p plugin) Init() {
}
