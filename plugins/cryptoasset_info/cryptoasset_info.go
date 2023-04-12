/*
cryptoasset_info is a gofra plugin that provides a brief description of crypto currency assets
*/

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/XaviFP/gofra/internal"
)

var Plugin plugin

type plugin struct{}

const metadataPrefix = "https://api.cryptowat.ch/assets/"
const metadataSufix = "/metadata"
const defaultAsset = "btc"

var g *gofra.Gofra

func (p plugin) Name() string {
	return "assetInfo"
}

func (p plugin) Description() string {
	return "Provides a brief description of crypto assets"
}

func (p plugin) Help() string {
	reply := g.Publish(gofra.Event{Name: "command/getCommandChar", MB: gofra.MessageBody{}, Payload: nil})
	commandChar := reply.GetAnswer()
	return fmt.Sprintf("Usage: %sassetinfo btc", commandChar)
}

func (p plugin) Init(conf gofra.Config, gofra *gofra.Gofra) {
	g = gofra

	g.Subscribe(
		"command/assetinfo",
		p.Name(),
		handleAssetInfo,
		0,
	)
}

func handleAssetInfo(e gofra.Event) *gofra.Reply {
	var asset string

	var r *gofra.Reply
	args := strings.Fields(e.MB.Body)[1:]
	argLength := len(args)

	switch {
	case argLength > 1:
		if err := g.SendStanza(e.MB.Reply("Too many arguments")); err != nil {
			g.Logger.Error(err.Error())
		}

		return r
	case argLength == 1:
		asset = args[0]
	default:
		asset = defaultAsset
	}

	resp, err := http.Get(metadataPrefix + asset + metadataSufix)
	if err != nil {
		g.Logger.Error(err.Error())
		if err := g.SendStanza(e.MB.Reply(fmt.Sprintf("Could not retrieve asset info: %s", err.Error()))); err != nil {
			g.Logger.Error(err.Error())

		}

		return r
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Could not retrieve asset info. Status code: %d", resp.StatusCode)
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

	payload, ok := result["result"][asset]
	if !ok {
		if err := g.SendStanza(e.MB.Reply("Asset not found")); err != nil {
			g.Logger.Error(err.Error())
		}

		return r
	}

	assetData := payload.(map[string]interface{})
	description, ok := assetData["AssetDescription"]
	if !ok {
		if err := g.SendStanza(e.MB.Reply("No description for " + asset + " yet")); err != nil {
			g.Logger.Error(err.Error())
		}

		return r
	}

	descriptionStr := description.(string)
	if descriptionStr == "" {
		if err := g.SendStanza(e.MB.Reply("No description for " + asset + " yet")); err != nil {
			g.Logger.Error(err.Error())
		}

		return r
	}

	if err := g.SendStanza(e.MB.Reply(descriptionStr)); err != nil {
		g.Logger.Error(err.Error())
	}

	return r
}
