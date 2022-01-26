package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/juju/errors"
)

type roundRequest struct {
	categories []int
	limit      int
}

type repository interface {
	GetRounds(roundRequest) ([]*round, error)
	GetCategories() ([]category, error)
}

func newOTDRepository() repository {
	return &otdRepo{url: "https://opentdb.com/api.php"}
}

type category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type otdRepo struct {
	url string
}

func (r *otdRepo) GetRounds(req roundRequest) ([]*round, error) {
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

	payload := struct {
		ResponseCode int      `json:"response_code"`
		Results      []*round `json:"results"`
	}{}

	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return []*round{}, err
	}

	return payload.Results, nil
}

func (r *otdRepo) GetCategories() ([]category, error) {
	res, err := http.Get("https://opentdb.com/api_category.php")
	if err != nil {
		return []category{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return []category{}, fmt.Errorf("http: status code: %d", res.StatusCode)
	}

	payload := struct {
		Categories []category `json:"trivia_categories"`
	}{}

	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return []category{}, err
	}

	return payload.Categories, nil
}
