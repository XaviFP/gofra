package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/juju/errors"

	"gofra/gofra"
)

var Plugin plugin

var errNoRounds = errors.New("no rounds available")

var g *gofra.Gofra
var session *gameSession
var repo repository

type plugin struct{}

func (p plugin) Name() string {
	return "trivia"
}

func (p plugin) Description() string {
	return "Trivia plugin"
}

func (p plugin) Init(conf gofra.Config, api *gofra.Gofra) {
	g = api
	g.Subscribe(
		"messageReceived",
		p.Name(),
		handleMessage,
		9999,
	)
	g.Subscribe(
		fmt.Sprintf("command/%s", p.Name()),
		p.Name(),
		handleCommand,
		9999,
	)

	repo = newOTDRepository()
	session = new(gameSession)
}

func StartNewSession(req roundRequest) error {
	rounds, err := repo.GetRounds(req)
	if err != nil {
		return errors.Annotate(err, "fetching rounds")
	}

	session = &gameSession{started: true}
	session.init(rounds)

	return nil
}

func handleCommand(e gofra.Event) *gofra.Reply {
	args := strings.Fields(e.MB.Body)[1:]
	r := e.MB.Reply

	if len(args) == 0 {
		g.SendStanza(r(`Use "!trivia start" to start a new game`))

		return nil
	}

	switch args[0] {
	case "start":
		if len(args) == 1 {
			if !session.started || session.finished {
				if err := StartNewSession(roundRequest{categories: []int{}, limit: 10}); err != nil {
					g.SendStanza(r(fmt.Sprintf("Could not start new session: %s", "a")))
				}

				g.SendStanza(r(session.current.String()))
			}

		} else {
			categoryID, err := strconv.Atoi(args[1])
			if err != nil {
				g.SendStanza(r("invalid category id"))

				return nil
			}

			StartNewSession(roundRequest{categories: []int{categoryID}, limit: 10})
			g.SendStanza(r(session.current.String()))
		}

	case "categories":
		res, err := repo.GetCategories()
		if err != nil {
			g.SendStanza(r(
				fmt.Sprintf("could not retrieve categories: %s", err),
			))

			return nil
		}

		var categories string
		for _, c := range res {
			categories = fmt.Sprintf("%s%d: %s\n", categories, c.ID, c.Name)
		}

		g.SendStanza(r(categories))
	}

	return nil
}

func handleMessage(e gofra.Event) *gofra.Reply {
	nextQuestion, ok := processRound(e.MB.From.Resourcepart(), e.MB.Body)
	if !ok {
		return nil
	}

	g.SendStanza(e.MB.Reply(nextQuestion))

	return nil
}

func processRound(player, answer string) (string, bool) {
	if !session.started || session.finished {
		return "", false
	}

	if ok := session.completeCurrent(player, answer); !ok {
		return "", false
	}

	nextQuestion, err := session.next()
	if errors.Cause(err) == errNoRounds {
		session.finished = true
		return session.summary(), true
	}

	return nextQuestion, true
}
