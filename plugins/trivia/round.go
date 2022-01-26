package main

import (
	"fmt"
	"html"
	"math/rand"
	"time"
)

type completedRound struct {
	r      *round
	player string
}

type round struct {
	Category         string   `json:"category"`
	Question         string   `json:"question"`
	Type             string   `json:"type"`
	Difficulty       string   `json:"difficulty"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
	answers          map[string]string
}

func (r *round) String() string {
	return fmt.Sprintf("%s\n%s\n%s\n%s", r.Category, r.Difficulty, r.Question, r.formatAnswers())
}

func (r *round) init() {
	r.Question = html.UnescapeString(r.Question)
	r.CorrectAnswer = html.UnescapeString(r.CorrectAnswer)

	for i := range r.IncorrectAnswers {
		r.IncorrectAnswers[i] = html.UnescapeString(r.IncorrectAnswers[i])
	}

	r.randomize()
}

func (r *round) formatAnswers() string {
	out := fmt.Sprintf(
		"A) %s\nB) %s",
		r.answers["A"],
		r.answers["B"],
	)

	if r.Type == "multiple" {
		out = fmt.Sprintf(
			"%s\nC) %s\nD) %s",
			out,
			r.answers["C"],
			r.answers["D"],
		)
	}

	return out
}

func (r *round) randomize() {
	answers := append(r.IncorrectAnswers, r.CorrectAnswer)

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(answers), func(i, j int) {
		answers[i], answers[j] = answers[j], answers[i]
	})

	r.answers = make(map[string]string)
	r.answers["A"] = answers[0]
	r.answers["B"] = answers[1]

	if r.Type == "multiple" {
		r.answers["C"] = answers[2]
		r.answers["D"] = answers[3]
	}
}
