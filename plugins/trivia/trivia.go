package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/juju/errors"

	"gofra/gofra"

	"mellium.im/xmpp/stanza"
)

var Plugin plugin

var errNoRounds = errors.New("no rounds available")

var g *gofra.Gofra
var config gofra.Config
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
	config = conf
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

func handleCommand(e gofra.Event) gofra.Reply {
	msg, ok := e.GetStanza().(gofra.MessageBody)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", e.GetStanza())
		return gofra.Reply{Empty: true}
	}

	args := strings.Split(e.MB.Body, " ")[1:]

	switch args[0] {
	case "start":
		if len(args) == 1 {
			if !session.started || session.finished {
				if err := StartNewSession(roundRequest{categories: []int{}, limit: 10}); err != nil {
					g.SendStanza(msg.Reply(config, fmt.Sprintf("Could not start new session: %s", "a")))
				}

				g.SendStanza(msg.Reply(config, session.current.String()))
			}

		} else {
			categoryID, err := strconv.Atoi(args[1])
			if err != nil {
				g.SendStanza(msg.Reply(config, "invalid category id"))

				return gofra.Reply{Empty: true}
			}

			StartNewSession(roundRequest{categories: []int{categoryID}, limit: 10})
			g.SendStanza(msg.Reply(config, session.current.String()))
		}

	case "categories":
		res, err := repo.GetCategories()
		if err != nil {
			g.SendStanza(msg.Reply(
				config,
				fmt.Sprintf("could not retrieve categories: %s", err),
			))

			return gofra.Reply{Empty: true}
		}

		var categories string
		for _, c := range res {
			categories = fmt.Sprintf("%s%d: %s\n", categories, c.ID, c.Name)
		}

		g.SendStanza(msg.Reply(config, categories))
	}

	return gofra.Reply{Empty: true}
}

func handleMessage(e gofra.Event) gofra.Reply {
	msg, ok := e.GetStanza().(gofra.MessageBody)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", e.GetStanza())
		return gofra.Reply{Empty: true}
	}

	if msg.Type != stanza.GroupChatMessage {
		return gofra.Reply{Empty: true}
	}

	if msg.Body == "" {
		return gofra.Reply{Empty: true}
	}

	nextQuestion, ok := processRound(msg.From.Resourcepart(), msg.Body)
	if !ok {
		return gofra.Reply{Empty: true}
	}

	g.SendStanza(msg.Reply(config, nextQuestion))
	return gofra.Reply{Empty: true}

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
