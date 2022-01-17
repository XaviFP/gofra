package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gofra/gofra"

	"mellium.im/xmpp/stanza"
)

var Plugin plugin

type plugin struct {
	roundRepo roundRepository
	// session   *gameSession
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
	rand.Shuffle(len(answers), func(i, j int) { answers[i], answers[j] = answers[j], answers[i] })

	r.answers = make(map[string]string)
	switch r.Type {
	case "multiple":
		r.answers["A"] = answers[0]
		r.answers["B"] = answers[1]
		r.answers["C"] = answers[2]
		r.answers["D"] = answers[3]
	case "boolean":
		r.answers["A"] = answers[0]
		r.answers["B"] = answers[1]
	}
}

type completedRound struct {
	r      *round
	player string
}

type roundRequest struct {
	categories []int
	limit      int
	from       int
}

type category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type roundRepository interface {
	Get(roundRequest) ([]*round, error)
	GetCategories() ([]category, error)
}

func newOTDRoundRepository() roundRepository {
	return &roundOTDRoundRepo{url: "https://opentdb.com/api.php"}
}

type roundOTDRoundRepo struct {
	url string
}

func (r *roundOTDRoundRepo) Get(req roundRequest) ([]*round, error) {
	url := fmt.Sprintf("%s?amount=%d", r.url, req.limit)
	if len(req.categories) > 0 {
		url = fmt.Sprintf("%s&category=%d", url, req.categories[0])
	}

	res, err := http.Get(url)
	if err != nil {
		return []*round{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return []*round{}, errors.New("")
	}

	type otdbRes struct {
		ResponseCode int      `json:"response_code"`
		Results      []*round `json:"results"`
	}

	var payload otdbRes
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return []*round{}, err
	}

	return payload.Results, nil
}

func (r *roundOTDRoundRepo) GetCategories() ([]category, error) {
	res, err := http.Get("https://opentdb.com/api_category.php")
	if err != nil {
		return []category{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return []category{}, fmt.Errorf("http: status code: %d", res.StatusCode)
	}

	type otdbRes struct {
		Categories []category `json:"trivia_categories"`
	}

	var payload otdbRes
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return []category{}, err
	}

	return payload.Categories, nil
}

type gameSession struct {
	rounds    []*round
	current   *round
	completed []*completedRound
	started   bool
	finished  bool
}

func (r *round) init() {
	r.Question = html.UnescapeString(r.Question)
	r.CorrectAnswer = html.UnescapeString(r.CorrectAnswer)

	for i := range r.IncorrectAnswers {
		r.IncorrectAnswers[i] = html.UnescapeString(r.IncorrectAnswers[i])
	}

	r.randomize()
}

func (s *gameSession) init(rounds []*round) error {
	for _, r := range rounds {
		r.init()
	}

	s.current = rounds[0]
	s.rounds = rounds[1:]

	return nil
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

var ErrNoRounds = errors.New("trivia: no rounds available")

func (s *gameSession) next() (string, error) {
	if len(s.rounds) == 0 {
		return "", ErrNoRounds
	}

	s.current = s.rounds[0]
	s.rounds = s.rounds[1:]

	return s.current.String(), nil
}

var g *gofra.Gofra
var config gofra.Config
var session *gameSession
var repo roundRepository

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
		p.handleMessage,
		9999,
	)
	g.Subscribe(
		fmt.Sprintf("command/%s", p.Name()),
		p.Name(),
		p.handleCommand,
		9999,
	)

	// load trivia files
	repo = newOTDRoundRepository()
	session = new(gameSession)
}

func (p plugin) NewSession(req roundRequest) {
	rounds, err := repo.Get(req)
	if err != nil {
		g.Logger.Error(fmt.Sprintf("fetching rounds: %s", err))
		return
	}

	session = &gameSession{started: true}
	if err := session.init(rounds); err != nil { // TODO returns no error, fix this
		g.Logger.Error(fmt.Sprintf("initializing game session: %s", err))
		return
	}
}

func (p plugin) handleCommand(e gofra.Event) gofra.Reply {
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
				p.NewSession(roundRequest{categories: []int{}, limit: 10})
				g.SendStanza(msg.Reply(config, session.current.String()))
			}
		} else {
			categoryID, err := strconv.Atoi(args[1])
			if err != nil {
				g.SendStanza(msg.Reply(config, "invalid category id"))
				return gofra.Reply{Empty: true}
			}
			p.NewSession(roundRequest{categories: []int{categoryID}, limit: 10})
			g.SendStanza(msg.Reply(config, session.current.String()))
		}

	case "categories":
		categories, err := repo.GetCategories()
		if err != nil {
			g.Logger.Error(err.Error())
			str := fmt.Sprintf("could not retrieve categories: %s", err)
			g.SendStanza(msg.Reply(config, str))
			return gofra.Reply{Empty: true}
		}
		var out string
		for _, c := range categories {
			out = fmt.Sprintf("%s%d: %s\n", out, c.ID, c.Name)
		}

		g.SendStanza(msg.Reply(config, out))
	}

	return gofra.Reply{Empty: true}
}

func (p plugin) handleMessage(e gofra.Event) gofra.Reply {
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

	nextQuestion, ok := p.handleRound(msg.From.Resourcepart(), msg.Body)
	if !ok {
		return gofra.Reply{Empty: true}
	}

	g.SendStanza(msg.Reply(config, nextQuestion))
	return gofra.Reply{Empty: true}

}

func (p plugin) handleRound(player, answer string) (string, bool) {
	if !session.started {
		return "", false
	}

	if ok := session.completeCurrent(player, answer); !ok {
		return "", false
	}

	nextQuestion, err := session.next()
	if errors.Is(err, ErrNoRounds) {
		session.finished = true
		return session.summary(), true
	}

	return nextQuestion, true
}
