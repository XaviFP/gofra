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

type plugin struct{}

type throw struct {
	quantity int
	faces    int
}

const command = "dice"

var g *gofra.Gofra
var config gofra.Config
var defaultDice = 6

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
		throwDice,
		0,
	)
	dd, exists := config.Plugins["Dice"]["defaultDice"].(int)
	if exists && defaultDice != dd && dd >= 2  {
		defaultDice = config.Plugins["Dice"]["defaultDice"].(int)
	}
}

func parseArgs(argLine string) []throw {
	args := strings.Split(argLine, " ")
	//Remove command and leave just the args for it
	args = args[1:]
	throws := []throw{}
	for _, arg := range args {

		if arg == "" {
			continue
		}

		number, err := strconv.Atoi(arg)
		if err == nil {
			throws = append(throws, throw{quantity: number, faces: defaultDice})
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

func do(throw throw, r *rand.Rand) string {
	results := fmt.Sprintf("%dd%d: ", throw.quantity, throw.faces)

	for i := 0; i < throw.quantity-1; i++ {
		results += fmt.Sprintf("%d, ", r.Intn(throw.faces) + 1)
	}
	results += fmt.Sprintf("%d", r.Intn(throw.faces) + 1)

	return results
}

func throwDice(e gofra.Event) gofra.Reply {

	argLine := strings.Split(e.MB.Body, " ")
	if argLine[0] != config.Plugins["Commands"]["commandChar"].(string)+command {
		if err := g.SendStanza(e.MB.Reply("Wrong command")); err != nil {
			g.Logger.Error(err.Error())
			return gofra.Reply{}
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	throws := parseArgs(e.MB.Body)

	if len(throws) == 0 {
		if err := g.SendStanza(e.MB.Reply("Need dice information to throw")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{}
		}
	}

	answer := ""
	for _, throw := range throws {
		answer += do(throw, r) + "\n"
	}

	if err := g.SendStanza(e.MB.Reply(answer)); err != nil {
		g.Logger.Error(err.Error())

		return gofra.Reply{Empty: true}
	}

	return gofra.Reply{Ok: true}
}

var Plugin plugin
