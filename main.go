/*
gofra is an XMPP bot engine.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	"gopkg.in/yaml.v3"

	"github.com/XaviFP/gofra/gofra"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT and gracefully shut down the bot.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		select {
		case <-ctx.Done():
		case <-c:
			cancel()
		}
	}()

	g = gofra.NewGofra(ctx, config)

	defer func() {
		g.Logger.Info("Closing conn…")
		if err := g.Client.Conn().Close(); err != nil {
			g.Logger.Error(fmt.Sprintf("Error closing connection: %q", err))
		}
	}()

	go func() {
		<-ctx.Done()
		g.Logger.Info("Closing session…")

		if err := g.Client.Close(); err != nil {
			g.Logger.Error(fmt.Sprintf("Error closing session: %q", err))
		}
	}()

	err := g.Init()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = g.Connect()
	if err != nil {
		log.Fatal(err.Error())
	}
}
