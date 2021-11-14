/*
gofra is an XMPP bot engine.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"

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

type logWriter struct {
	logger *log.Logger
}

func (lw logWriter) Write(p []byte) (int, error) {
	lw.logger.Printf("%s", p)
	return len(p), nil
}

func getStreamLoggers(config gofra.Config) (io.Writer, io.Writer, log.Logger, log.Logger, error){
	// Setup logging and verbose logging that's disabled by default.
	logger := log.New(os.Stderr, "", log.LstdFlags)
	debug := log.New(io.Discard, "DEBUG ", log.LstdFlags)

	// Configure behavior based on config file.
	var (
		verbose bool
		logXML  bool
	)
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage of %s:\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\n  $%s: The JID which will be used to listen for messages to echo\n  $%s: The password\n\n", config.Jid, config.Password)
		flags.PrintDefaults()
	}
	flags.BoolVar(&verbose, "v", verbose, "turns on verbose debug logging")
	flags.BoolVar(&logXML, "vv", logXML, "turns on verbose debug and XML logging")

	switch err := flags.Parse(os.Args[1:]); err {
	case flag.ErrHelp:
		return nil, nil, *logger, *debug, fmt.Errorf("flag.ErrHelp: %v", err)
	case nil:
	default:
		logger.Fatal(err)
	}

	// Enable verbose logging if the flag was set.
	if verbose || logXML {
		debug.SetOutput(os.Stderr)
	}

	// Enable XML logging if the flag was set.
	var xmlIn, xmlOut io.Writer
	if logXML {
		xmlIn = logWriter{log.New(os.Stdout, "IN ", log.LstdFlags)}
		xmlOut = logWriter{log.New(os.Stdout, "OUT ", log.LstdFlags)}
	}
	return xmlIn, xmlOut, *logger, *debug, nil
}

func main() {
	xmlIn, xmlOut, logger, debug, err := getStreamLoggers(config)

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

	g = gofra.NewGofra(ctx, config, xmlIn, xmlOut, &logger, &debug)
	err = g.Init()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = g.Connect()
	if err != nil {
		log.Fatal(err.Error())
	}
}
