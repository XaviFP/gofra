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

var Plugin plugin
var g *gofra.Gofra

type plugin struct{}

func (p plugin) Name() string {
	return "Pick"
}

func (p plugin) Description() string {
	return "Picks among a given list"
}

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	g = gofra

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

		options = append(options, strings.Trim(arg, " \n\t"))
	}

	return quantity, options
}

func choose(quantity int, options []string, r *rand.Rand) string {
	out := "Chose: "

	if quantity >= len(options) {
		return out + "All the options"
	}

	if quantity == 1 {
		out += options[r.Intn(len(options))]

		return out
	}

	for i := 0; i < quantity-1; i++ {
		choice := r.Intn(len(options))
		out += options[choice]
		options = append(options[:choice], options[choice+1:]...)

		if i < quantity-2 {
			out += ","
		}

		out += " "
	}

	return fmt.Sprintf("%sand %s", out, options[r.Intn(len(options))])
}

func pick(e gofra.Event) *gofra.Reply {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	quantity, options := parseArgs(e.MB.Body)
	answer := choose(quantity, options, r)

	if err := g.SendStanza(e.MB.Reply(answer)); err != nil {
		g.Logger.Error(err.Error())

		return nil
	}

	return nil
}
