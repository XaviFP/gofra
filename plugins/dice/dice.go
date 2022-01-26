/*
dice is a gofra plugin that provides a utility to simulate dice throws
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
var config gofra.Config
var defaultDiceFaces = 6
var defaultDiceQuantity = 1

type throw struct {
	quantity int
	faces    int
}

type plugin struct{}

func (p plugin) Name() string {
	return "Dice"
}

func (p plugin) Description() string {
	return "Provides dice throwing results"
}

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	g = gofra
	config = c

	g.Subscribe(
		"command/dice",
		p.Name(),
		handleCommand,
		0,
	)

	df, exists := config.Plugins["Dice"]["faces"].(int)
	if exists && defaultDiceFaces != df && df >= 2 {
		defaultDiceFaces = config.Plugins["Dice"]["faces"].(int)
	}
	dq, exists := config.Plugins["Dice"]["quantity"].(int)
	if exists && defaultDiceQuantity != dq && dq >= 1 {
		defaultDiceQuantity = config.Plugins["Dice"]["quantity"].(int)
	}

}

func handleCommand(e gofra.Event) *gofra.Reply {
	throws := parseArgs(e.MB.Body)
	answer := ""
	for _, throw := range throws {
		answer += do(throw) + "\n"
	}

	if err := g.SendStanza(e.MB.Reply(answer)); err != nil {
		g.Logger.Error(err.Error())

		return nil
	}

	return nil
}

func parseArgs(argLine string) []throw {
	args := strings.Split(argLine, " ")[1:]

	if len(args) == 0 {
		return []throw{{quantity: defaultDiceQuantity, faces: defaultDiceFaces}}
	}

	throws := []throw{}
	for _, arg := range args {
		if arg == "" {
			throws = append(throws, throw{quantity: defaultDiceQuantity, faces: defaultDiceFaces})
		}

		number, err := strconv.Atoi(arg)
		if err == nil {
			throws = append(throws, throw{quantity: number, faces: defaultDiceFaces})

			continue
		}

		t := strings.Split(arg, "d")
		if len(t) != 2 {
			continue
		}

		number, err = strconv.Atoi(t[0])
		if err != nil {
			continue
		}

		faces, err := strconv.Atoi(t[1])
		if err != nil {
			continue
		}
		if faces < 2 {
			faces = 2
		}

		throws = append(throws, throw{quantity: number, faces: faces})
	}

	return throws
}

func do(throw throw) string {
	results := fmt.Sprintf("%dd%d: ", throw.quantity, throw.faces)

	for i := 0; i < throw.quantity-1; i++ {
		rand.Seed(time.Now().UnixNano())
		results += fmt.Sprintf("%d, ", rand.Intn(throw.faces)+1)
	}

	return fmt.Sprintf("%s%d", results, rand.Intn(throw.faces)+1)
}
