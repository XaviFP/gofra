/*
cryptoasset_info is a gofra plugin that provides a brief description of crypto currency assets
*/

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"gofra/gofra"
	"plugins/command"
)

type plugin string

const commandStr = "assetinfo"
const metadataPrefix = "https://api.cryptowat.ch/assets/"
const metadataSufix = "/metadata"
const defaultAsset = "btc"

var g gofra.API
var config gofra.Config

func (p plugin) Name() string {
	return "CryptoAssetInfo"
}

func (p plugin) Description() string {
	return "Provides a brief description of crypto assets"
}

func (p plugin) Init(conf gofra.Config, api gofra.API) {
	g = api
	config = conf
	g.Subscribe(
		"command/assetinfo",
		p.Name(),
		handleAssetInfo,
		0,
	)
}

func handleAssetInfo(e gofra.Event) gofra.Reply {
	var r gofra.Reply
	asset := defaultAsset
	argLine := e.Payload["commandBody"].(string)
	args := strings.Split(argLine, " ")
	if args[0] != config.Plugins["Commands"]["commandChar"].(string)+commandStr {
		r = gofra.Reply{}
		r.SetAnswer("Wrong command")
		return r
	}
	//Remove command and leave just the args for it
	args = args[1:]
	if argLine != "" {
		if len(args) > 1 {
			//return "Too many arguments"
		} else if len(args) == 1 {
			asset = args[0]
		}
	}
	resp, err := http.Get(metadataPrefix + asset + metadataSufix)

	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("Something went wrong")
		return r
	}
	var result map[string]interface{}
	log.Println(resp.Body)
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("Invalid response")
		return r
	}

	aux := result["result"]
	resultMap := aux.(map[string]interface{})
	payload, ok := resultMap[asset]
	if !ok {
		r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("Asset not found")
		return r
	}
	payloadMap := payload.(map[string]interface{})
	description, ok := payloadMap["AssetDescription"]
	if !ok {
		r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("No description for " + asset + " yet")
		return r
	}
	descriptionString := description.(string)
	if descriptionString == "" {
		r = gofra.Reply{Ok: true, Empty: false}
		r.SetAnswer("No description for " + asset + " yet")
		return r
	}
	log.Println(result)

	r = gofra.Reply{Ok: true, Empty: false}
	r.SetAnswer(descriptionString)

	command.YOLO()

	return r
}

var Plugin plugin
