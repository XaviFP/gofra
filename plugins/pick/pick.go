/*
pick is a gofra plugin that chooses randomly an element (or elements) from a provided list
*/

package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"gofra/gofra"
)

type plugin struct{}

const command = "pick"

var g *gofra.Gofra
var config gofra.Config

func (p plugin) Name() string {
	return "Pick"
}

func (p plugin) Description() string {
	return "Picks among a given list"
}

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	g = gofra
	config = c
	g.Subscribe(
		"command/pick",
		p.Name(),
		pick,
		0,
	)
}

func parseArgs(argLine string) (int, []string) {
	args := strings.Split(argLine, " ")
	command := args[0]
	optLine := ""
	//Remove command and leave just the args for it
	args = args[1:]
	quantity, err := strconv.Atoi(args[0])
	if err != nil || quantity < 1 {
		quantity = 1
		optLine = argLine[len(command)+1:]
	} else {
		optStart := len(command) + 1 + len(args[0]) + 1
		optLine = argLine[optStart:]
	}

	options := []string{}
	for _, arg := range strings.Split(optLine, ",") {

		if arg == "" {
			continue
		}

		option := strings.Trim(arg, " \n\t")
		options = append(options, option)
	}

	return quantity, options
}

func choose(quantity int, options []string, r *rand.Rand) string {
	choices := "Chose: "

	if quantity >= len(options) {

		return choices + "All the options"
	}

	if quantity == 1 {
		choices += options[r.Intn(len(options))]

		return choices
	}

	for i := 0; i < quantity-1; i++ {
		choice := r.Intn(len(options))
		choices += options[choice]
		options = append(options[:choice], options[choice+1:]...)
		if i < quantity-2 {
			choices += ","
		}
		choices += " "
	}
	choices += fmt.Sprintf("and %s", options[r.Intn(len(options))])

	return choices
}

func pick(e gofra.Event) *gofra.Reply {

	argLine := strings.Split(e.MB.Body, " ")
	if argLine[0] != config.Plugins["Commands"]["commandChar"].(string)+command {
		if err := g.SendStanza(e.MB.Reply("Wrong command")); err != nil {
			g.Logger.Error(err.Error())
			return nil
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	quantity, options := parseArgs(e.MB.Body)
	answer := choose(quantity, options, r)

	if err := g.SendStanza(e.MB.Reply(answer)); err != nil {
		g.Logger.Error(err.Error())

		return nil
	}

	return &gofra.Reply{Ok: true}
}

var Plugin plugin
