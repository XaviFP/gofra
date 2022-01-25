/*
session_tracker is a gofra plugin that allows users to keep track of tasks done during a working session
*/

package main

import (
	"strings"
	"time"

	"gofra/gofra"
)

var Plugin plugin

var g *gofra.Gofra

type plugin struct{}

func (p plugin) Name() string {
	return "SessionTracker"
}

func (p plugin) Description() string {
	return "Keeps track of tasks done in a working session"
}

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	g = gofra
	g.Subscribe(
		"command/st",
		p.Name(),
		handleSession,
		0,
	)
}

func handleSession(e gofra.Event) *gofra.Reply {
	args := strings.Split(e.MB.Body, " ")[1:]

	s, exists := sessions[e.MB.From.String()]
	if len(args) < 1 {
		if !exists || s.status == NoSession {
			if err := g.SendStanza(e.MB.Reply("You don't have an ongoing session")); err != nil {
				g.Logger.Error(err.Error())
			}

			return nil
		}

		s.update()
		sessions[e.MB.From.String()] = s

		if err := g.SendStanza(e.MB.Reply(s.String())); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil
	}

	command := args[0]

	if (!exists || s.status == NoSession) && command != "start" {
		if err := g.SendStanza(e.MB.Reply("You don't have an ongoing session")); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil
	}

	switch args[0] {
	case "start":
		if s.status != NoSession {
			if err := g.SendStanza(e.MB.Reply("You already have an ongoing session")); err != nil {
				g.Logger.Error(err.Error())
			}
		}

		sessions[e.MB.From.String()] = session{
			status:     Running,
			tasks:      []task{},
			startedAt:  time.Now(),
			duration:   time.Duration(0),
			lastUpdate: time.Now(),
		}

		if err := g.SendStanza(e.MB.Reply("Session started!")); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil

	case "pause":
		if s.status == Paused {
			if err := g.SendStanza(e.MB.Reply("Session is already paused")); err != nil {
				g.Logger.Error(err.Error())
			}

			return nil
		}

		s.pause()
		sessions[e.MB.From.String()] = s

		if err := g.SendStanza(e.MB.Reply("Session paused")); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil

	case "resume":
		if s.status == Running {
			if err := g.SendStanza(e.MB.Reply("Session is already running")); err != nil {
				g.Logger.Error(err.Error())
			}

			return nil
		}

		s.resume()
		sessions[e.MB.From.String()] = s

		if err := g.SendStanza(e.MB.Reply("Session is running again")); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil

	case "stop":
		s.update()
		s.status = Stopped
		if err := g.SendStanza(e.MB.Reply(s.String())); err != nil {
			g.Logger.Error(err.Error())
		}

		s.stop()
		sessions[e.MB.From.String()] = s

		return nil

	case "add":
		description := strings.Join(args[1:], " ")
		session := sessions[e.MB.From.String()]
		session.tasks = append(session.tasks, task{description: description, time: time.Now()})
		sessions[e.MB.From.String()] = session

		if err := g.SendStanza(e.MB.Reply("Task added")); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil

	default:
		if err := g.SendStanza(e.MB.Reply("Session tracker subcommand not recognized.\nTry with start, pause, resume or stop")); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil
	}
}
