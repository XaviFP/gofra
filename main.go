/*
gofra is an XMPP bot engine.
*/

package main

import (
	gofra "gofra/gofra"
	"log"
)

// Configuration options that will be set when deploying.
// Will be loaded from deployment/config file
var config = gofra.Config{
	ServerURL: "blastersklan.com",
	ServerPort: "5222",
	Password: "1234",
	Plugins_paths: []string{"plugins/"},
	Jid: "golang@blastersklan.com",
	Nick: "Gofra",
	Mucs: []gofra.MucConfig{
		{Nick: "gofra",
		MucJoinHistory: 0,
		MucJid: "shigoto@agora.blastersklan.com",},
	},
	MucJoinHistory: 0,
	Extra: make(map[string]interface{}),
}

var g *gofra.Gofra
func main() {
	conf, err := getConfig()
	if err != nil {
		// Log wrong config and exit
		log.Fatal(err.Error())
	}
	
	g = gofra.NewGofra(conf)
	err = g.Init()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = g.Connect()
	if err != nil {
		log.Fatal(err.Error())
	}
	// Auto re-connect etc
	for{}
}

func getConfig() (gofra.Config, error) {
	//config := commons.Config{}
	// . . .
	// . . .
	return config, nil
}
