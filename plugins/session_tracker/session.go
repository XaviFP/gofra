package main

import (
	"fmt"
	"strings"
	"time"
)

type sessionStatus int

const (
	NoSession sessionStatus = iota
	Running
	Paused
	Stopped
)

func (s sessionStatus) String() string {
	switch s {
	case NoSession:
		return "No Session"
	case Running:
		return "Running"
	case Paused:
		return "Paused"
	case Stopped:
		return "Stopped"
	}

	return ""
}

type task struct {
	description string
	time        time.Time
}

var sessions = make(map[string]session)

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
	out := fmt.Sprintf(
		"Session status: %s\nStarted at: %v\nDuration: %s\n",
		s.status,
		s.startedAt.Format("2006-Jan-02 03:04:05 PM"),
		s.duration.Round(time.Second),
	)

	if len(s.tasks) > 0 {
		t := []string{"Tasks during session:\n"}
		for i, task := range s.tasks {
			t = append(t, fmt.Sprintf(
				"%d- %s. Started at: %v\n",
				i+1,
				task.description,
				task.time.Format("2006-Jan-02 03:04:05 PM"),
			))
		}

		out = out + strings.Join(t, "")
	}

	return out
}
