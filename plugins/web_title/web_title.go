package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	gofra "github.com/XaviFP/gofra/internal"
)

var Plugin plugin

var g *gofra.Gofra

type plugin struct{}

func (p plugin) Name() string {
	return "Web title"
}

func (p plugin) Description() string {
	return "Parses urls and writes a message back with the website's title"
}

func (p plugin) Help() string {
	return "Writes back title of websites if message contains url and website's url has a title"
}

func (p plugin) Init(config gofra.Config, gofra *gofra.Gofra) {
	g = gofra

	g.Subscribe(
		"messageReceived",
		p.Name(),
		handleMessage,
		1,
	)
}

func handleMessage(e gofra.Event) *gofra.Reply {
	// Parse e.MB.Body to see if it contains a URL
	url := containsURL(e.MB.Body)
	if url == "" {
		g.Logger.Error("no url found in message")
		return nil
	}
	g.Logger.Error(fmt.Sprintf("url in message: %s", url))
	title, err := getTitle(url)
	if err != nil {
		g.Logger.Error(fmt.Sprintf("no title couldn't be retrieved, error: %s", err))
		return nil
	}
	g.Logger.Error(fmt.Sprintf("title found for url: %s", title))

	if err := g.SendStanza(e.MB.Reply(title)); err != nil {
		g.Logger.Error(err.Error())

		return nil
	}

	return nil
}

func containsURL(text string) string {
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	url := urlRegex.FindString(text)
	return url
}
func getTitle(url string) (string, error) {
	// Make the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Read the body of the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Define a regex pattern to find the <title> tag content
	titleRegex := regexp.MustCompile(`<title>(.*?)</title>`)
	matches := titleRegex.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("no <title> tag found")
	}

	// Return the content of the <title> tag
	return matches[1], nil
}
