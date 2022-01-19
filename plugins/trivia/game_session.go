package main

import (
	"fmt"
	"strings"
)

type gameSession struct {
	rounds    []*round
	current   *round
	completed []*completedRound
	started   bool
	finished  bool
}

func (s *gameSession) init(rounds []*round) {
	for _, r := range rounds {
		r.init()
	}

	s.current = rounds[0]
	s.rounds = rounds[1:]
}

func (s *gameSession) completeCurrent(player, answer string) bool {
	if strings.EqualFold(s.current.CorrectAnswer, s.current.answers[strings.ToUpper(answer)]) {
		s.completed = append(s.completed, &completedRound{r: s.current, player: player})
		return true
	}

	return false
}

func (s *gameSession) summary() string {
	length := len(s.completed)
	scoreboard := make(map[string]int, length)

	for _, r := range s.completed {
		if _, exists := scoreboard[r.player]; !exists {
			scoreboard[r.player] = 1
		} else {
			scoreboard[r.player] = scoreboard[r.player] + 1
		}
	}

	summary := "Results:\n"

	for player, score := range scoreboard {
		summary = fmt.Sprintf("%s%s\t%d/%d\n", summary, player, score, length)
	}

	return summary
}

func (s *gameSession) next() (string, error) {
	if len(s.rounds) == 0 {
		return "", errNoRounds
	}

	s.current = s.rounds[0]
	s.rounds = s.rounds[1:]

	return s.current.String(), nil
}
