/*
pairs_price is a gofra plugin that provides an api to check cryptocurrency pair prices
*/

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gofra/gofra"
)

type plugin string

const command = "price"
const metadataPrefix = "https://api.cryptowat.ch/markets/"
const metadataSufix = "/price"
const defaultExchange = "kraken"
const defaultPair = "btcusd"

var g gofra.Gofra
var config gofra.Config

func (p plugin) Name() string {
	return "Price"
}

func (p plugin) Description() string {
	return "Provides price equivalences of crypto assets"
}

func (p plugin) Init(c gofra.Config, gofra gofra.Gofra) {
	g = gofra
	config = c
	g.Subscribe(
		"command/price",
		p.Name(),
		handlePrice,
		0,
	)
}

func handlePrice(e gofra.Event) gofra.Reply {
	exchange := defaultExchange
	pair := defaultPair
	argLine := e.MB.Body
	args := strings.Split(argLine, " ")
	if args[0] != config.Plugins["Commands"]["commandChar"].(string)+command {
		if err := g.SendStanza(e.MB.Reply("Too many arguments")); err != nil {
			g.Logger.Error(err.Error())
			return gofra.Reply{Ok: false}
		}
	}

	//Remove command and leave just the args for it
	args = args[1:]
	if argLine != "" {
		if len(args) > 2 {
			if err := g.SendStanza(e.MB.Reply("Too many arguments")); err != nil {
				g.Logger.Error(err.Error())

				return gofra.Reply{Empty: true}
			}

			return gofra.Reply{Ok: true}
		} else if len(args) == 2 {
			pair = args[0]
			exchange = args[1]
		} else if len(args) == 1 && args[0] != "" {
			pair = args[0]
		}
	}
	requestUrl := metadataPrefix + exchange + "/" + pair + metadataSufix
	log.Println(requestUrl)

	resp, err := http.Get(metadataPrefix + exchange + "/" + pair + metadataSufix)

	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	log.Println(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return gofra.Reply{Empty: true}
	}

	var result map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return gofra.Reply{Empty: true}
	}

	aux := result["result"]
	resultMap := aux.(map[string]interface{})
	priceField, ok := resultMap["price"]
	if !ok {
		if err := g.SendStanza(e.MB.Reply("Price for pair not found")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{Empty: true} // TODO LOG ERROR
		}

		return gofra.Reply{Ok: true}
	}

	priceFloat := priceField.(float64)
	price := strconv.FormatFloat(priceFloat, 'f', -1, 64)

	log.Println(price)

	if err := g.SendStanza(e.MB.Reply(price)); err != nil {
		g.Logger.Error(err.Error())

		return gofra.Reply{Empty: true} // TODO LOG ERROR
	}

	return gofra.Reply{Ok: true}
}

var Plugin plugin
