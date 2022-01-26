/*
pairs_price is a gofra plugin that provides an api to check cryptocurrency pair prices
*/

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gofra/gofra"
)

var Plugin plugin

const metadataPrefix = "https://api.cryptowat.ch/markets/"
const metadataSufix = "/price"
const defaultExchange = "kraken"
const defaultPair = "btcusd"

var g *gofra.Gofra

type plugin struct{}

func (p plugin) Name() string {
	return "Price"
}

func (p plugin) Description() string {
	return "Provides price equivalences of crypto assets"
}

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	g = gofra

	g.Subscribe(
		"command/price",
		p.Name(),
		handlePrice,
		0,
	)
}

func handlePrice(e gofra.Event) *gofra.Reply {
	var exchange, pair string

	var r *gofra.Reply
	args := strings.Split(e.MB.Body, " ")[1:]
	argLength := len(args)

	switch {
	case argLength > 2:
		if err := g.SendStanza(e.MB.Reply("Too many arguments")); err != nil {
			g.Logger.Error(err.Error())
		}

		return r
	case argLength == 2:
		exchange = args[1]
		pair = args[0]
	case argLength == 1:
		exchange = defaultExchange
		pair = args[0]
	default:
		exchange = defaultExchange
		pair = defaultPair
	}

	resp, err := http.Get(metadataPrefix + exchange + "/" + pair + metadataSufix)
	if err != nil {
		g.Logger.Error(err.Error())
		if err := g.SendStanza(e.MB.Reply(fmt.Sprintf("Could not retrieve asset price: %s", err.Error()))); err != nil {
			g.Logger.Error(err.Error())
		}

		return r
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Could not retrieve asset price. Status code: %d", resp.StatusCode)
		if err := g.SendStanza(e.MB.Reply(errMsg)); err != nil {
			g.Logger.Error(err.Error())
		}

		return r
	}

	var result map[string]map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if err := g.SendStanza(e.MB.Reply(fmt.Sprintf("Could not decode response: %s", err.Error()))); err != nil {
			g.Logger.Error(err.Error())
		}

		return r
	}

	priceField, ok := result["result"]["price"]
	if !ok {
		if err := g.SendStanza(e.MB.Reply("Price for pair not found")); err != nil {
			g.Logger.Error(err.Error())
		}

		return r
	}

	priceFloat := priceField.(float64)
	price := strconv.FormatFloat(priceFloat, 'f', -1, 64)

	if err := g.SendStanza(e.MB.Reply(fmt.Sprintf("%s: %s", pair, price))); err != nil {
		g.Logger.Error(err.Error())
	}

	return r
}
