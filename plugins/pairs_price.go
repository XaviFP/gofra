/*
pairs_price is a plugin for botname that provides an api to check cryptocurrency pair prices
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
const commandStr = "price"
const metadataPrefix = "https://api.cryptowat.ch/markets/"
const metadataSufix = "/price"
const defaultExchange = "kraken"
const defaultPair = "btcusd"

var g gofra.API
var config gofra.Config

func (p plugin) Name() string {
	return "Price"
}

func (p plugin) Description() string {
	return "Provides price equivalences of crypto assets"
}

func (p plugin) Init(c gofra.Config, api gofra.API) {
	g = api
	config = c
	g.Subscribe(
		"command/price",
		p.Name(),
		handlePrice,
		gofra.Options{},
	)
}

func handlePrice(e gofra.Event, _ *gofra.Event) (gofra.Reply, gofra.Event){
	var r gofra.Reply
	exchange := defaultExchange
	pair := defaultPair
	argLine := e.Payload["commandBody"].(string)
	args := strings.Split(argLine, " ")
	if args[0] != config.Plugins["Commands"]["commandChar"].(string) + commandStr {
		r = gofra.Reply{Ok: false, Empty: false}
		r.SetAnswer("Wrong command")
		return r, e 
	}
	
	//Remove command and leave just the args for it
	args = args[1:]
	if argLine != "" {
		if len(args) > 2 {
			r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("Too many arguments")
		return r, e
		} else if len(args) == 2 {
			pair = args[0]
			exchange = args[1]
		} else if len(args) == 1 && args[0] != "" {
			pair = args[0]
		}
	}
	requestUrl := metadataPrefix + exchange + "/" + pair + metadataSufix
	log.Println(requestUrl)
	resp, err := http.Get(requestUrl)

	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	log.Println(resp.Body)
	if resp.StatusCode != http.StatusOK {
		r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("Something went wrong")
		return r, e
	}
	var result map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("Invalid response")
		return r, e
	}

	aux := result["result"]
	resultMap := aux.(map[string]interface{})
	priceField, ok := resultMap["price"]
	if !ok {
		r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("Price for pair not found")
		return r, e
	}
	priceFloat := priceField.(float64)
	price := strconv.FormatFloat(priceFloat, 'f', -1, 64)

	log.Println(price)

	r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer(price)
		return r, e
}

var Plugin plugin
