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
)

type plugin struct{}

const commandStr = "assetinfo"
const metadataPrefix = "https://api.cryptowat.ch/assets/"
const metadataSufix = "/metadata"
const defaultAsset = "btc"

var g *gofra.Gofra
var config gofra.Config

func (p plugin) Name() string {
	return "CryptoAssetInfo"
}

func (p plugin) Description() string {
	return "Provides a brief description of crypto assets"
}

func (p plugin) Init(conf gofra.Config, gofra *gofra.Gofra) {
	g = gofra
	config = conf
	g.Subscribe(
		"command/assetinfo",
		p.Name(),
		handleAssetInfo,
		0,
	)
}

func handleAssetInfo(e gofra.Event) gofra.Reply {
	asset := defaultAsset
	argLine := e.MB.Body
	args := strings.Split(argLine, " ")
	if args[0] != config.Plugins["Commands"]["commandChar"].(string)+commandStr {
		if err := g.SendStanza(e.MB.Reply("Wrong command")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{}
		}
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
		if err := g.SendStanza(e.MB.Reply("Something went wrong")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{Ok: true, Empty: false}
		}
	}

	var result map[string]interface{}
	log.Println(resp.Body)
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if err := g.SendStanza(e.MB.Reply("Invalid response")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{Ok: true, Empty: false}
		}
	}

	aux := result["result"]
	resultMap := aux.(map[string]interface{})
	payload, ok := resultMap[asset]
	if !ok {
		if err := g.SendStanza(e.MB.Reply("Asset not found")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{Ok: true, Empty: false}
		}
	}

	payloadMap := payload.(map[string]interface{})
	description, ok := payloadMap["AssetDescription"]
	if !ok {
		if err := g.SendStanza(e.MB.Reply("No description for " + asset + " yet")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{Ok: true, Empty: false}
		}
	}

	descriptionString := description.(string)
	if descriptionString == "" {
		if err := g.SendStanza(e.MB.Reply("No description for " + asset + " yet")); err != nil {
			g.Logger.Error(err.Error())

			return gofra.Reply{Ok: true, Empty: false}
		}
	}

	log.Println(result)

	if err := g.SendStanza(e.MB.Reply(descriptionString)); err != nil {
		g.Logger.Error(err.Error())
	}

	return gofra.Reply{Ok: true, Empty: false}
}

var Plugin plugin
