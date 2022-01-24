/*
session_tracker is a gofra plugin that allows users to keep track of tasks done during a working session
*/

package main

import (
	"fmt"
	"strings"
	"time"

	"gofra/gofra"
)

var Plugin plugin

type plugin struct{}

type sessionStatus int

const (
	NoSession sessionStatus = iota
	Running
	Paused
	Stopped
)

var sessionStatusses map[sessionStatus]string = map[sessionStatus]string{NoSession: "No Session", Running: "Running", Paused: "Paused", Stopped: "Stopped"}

type session struct {
	status     sessionStatus
	tasks      []task
	startedAt  time.Time
	duration   time.Duration
	lastUpdate time.Time
}

func (s *session) update() {
	if s.status == Running {
		s.duration += time.Since(s.lastUpdate)
	}
	s.lastUpdate = time.Now()
}

func (s *session) pause() {
	s.update()
	s.status = Paused
}

func (s *session) resume() {
	s.status = Running
	s.lastUpdate = time.Now()
}

func (s *session) stop() {
	s.status = NoSession
	s.lastUpdate = time.Now()
}

func (s *session) String() string {
	r := fmt.Sprintf(
		"Session status: %s\nStarted at: %v\nCurrent duration: %s\n",
		sessionStatusses[s.status],
		s.startedAt.Format("2006-Jan-02 03:04:05 PM"),
		s.duration.Round(time.Second),
	)

	t := []string{"Tasks during session:\n"}
	for i, task := range s.tasks {
		t = append(t, fmt.Sprintf("%d- %s. Started at: %v\n", i+1, task.description, task.time))
	}
	return r + strings.Join(t, "")
}

func (s *session) StoppingString() string {
	r := fmt.Sprintf(
		"Session status: %s\nStarted at: %v\nTotal session duration: %s\n",
		sessionStatusses[s.status],
		s.startedAt.Format("2006-Jan-02 03:04:05 PM"),
		s.duration.Round(time.Second))
	t := []string{"Tasks during session:\n"}
	for i, task := range s.tasks {
		t = append(t, fmt.Sprintf("%d- %s. Started at: %v\n", i+1, task.description, task.time))
	}
	return r + strings.Join(t, "")
}

type task struct {
	description string
	time        time.Time
}

var g *gofra.Gofra

var sessions = make(map[string]session)

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
	//Remove command and leave just the args for it
	args := strings.Split(e.MB.Body, " ")[1:]

	s, exists := sessions[e.MB.From.String()]
	if len(args) < 1 || (len(args) > 0 && args[0] == "") {
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

		return &gofra.Reply{Ok: true}
	}

	if args[0] == "start" {
		if !exists || s.status == NoSession {
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

			return &gofra.Reply{Ok: true}
		}

		if err := g.SendStanza(e.MB.Reply("You already have an ongoing session")); err != nil {
			g.Logger.Error(err.Error())
		}

		return &gofra.Reply{Ok: true}
	}

	if !exists || s.status == NoSession {
		if err := g.SendStanza(e.MB.Reply("You don't have an ongoing session")); err != nil {
			g.Logger.Error(err.Error())
		}

		return &gofra.Reply{Ok: true}
	}

	switch c := args[0]; c {
	case "pause":
		if s.status == Paused {
			if err := g.SendStanza(e.MB.Reply("Session is already paused")); err != nil {
				g.Logger.Error(err.Error())
			}

			return &gofra.Reply{Ok: true}
		}

		s.pause()
		sessions[e.MB.From.String()] = s

		if err := g.SendStanza(e.MB.Reply("Session paused")); err != nil {
			g.Logger.Error(err.Error())
		}

		return &gofra.Reply{Ok: true}

	case "resume":
		if s.status == Running {
			if err := g.SendStanza(e.MB.Reply("Session is already running")); err != nil {
				g.Logger.Error(err.Error())
			}

			return &gofra.Reply{Ok: true}
		}

		s.resume()
		sessions[e.MB.From.String()] = s

		if err := g.SendStanza(e.MB.Reply("Session is running again")); err != nil {
			g.Logger.Error(err.Error())
		}

		return &gofra.Reply{Ok: true}

	case "stop":
		s.update()
		s.status = Stopped
		if err := g.SendStanza(e.MB.Reply(s.StoppingString())); err != nil {
			g.Logger.Error(err.Error())
		}

		s.stop()
		sessions[e.MB.From.String()] = s

		return &gofra.Reply{Ok: true}

	case "add":
		description := strings.Join(args[1:], " ")
		session := sessions[e.MB.From.String()]
		session.tasks = append(session.tasks, task{description: description, time: time.Now()})
		sessions[e.MB.From.String()] = session
		if err := g.SendStanza(e.MB.Reply("Task added")); err != nil {
			g.Logger.Error(err.Error())
		}

		return &gofra.Reply{Ok: true}

	default:
		if err := g.SendStanza(e.MB.Reply("Session tracker subcommand not recognized.\nTry with start, pause, resume or stop")); err != nil {
			g.Logger.Error(err.Error())
		}

		return nil
	}

}
