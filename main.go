/*
gofra is an XMPP bot engine.
*/

package main

import (
	"flag"
	"io/ioutil"
	"log"

	gofra "gofra/gofra"

	"gopkg.in/yaml.v3"
)

var config gofra.Config
var g *gofra.Gofra

func init() {
	configFilePathPtr := flag.String("config", "config.yaml", "file path of the config.yml file")
	flag.Parse()

	loadConfig(*configFilePathPtr)
}

func loadConfig(configFilePath string) {
	yamlFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln("Error reading config file", err)
	}

	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}

func main() {
	g = gofra.NewGofra(config)
	err := g.Init()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = g.Connect()
	if err != nil {
		log.Fatal(err.Error())
	}
	// Auto re-connect etc
	for {
	}
}
