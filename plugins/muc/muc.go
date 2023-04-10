/*
muc is a gofra plugin that allows joining muti-user chatrooms and keeps track of them
*/

package main

import (
	"fmt"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/stanza"

	"github.com/XaviFP/gofra/gofra"
)

var Plugin plugin

type plugin struct{}

var g *gofra.Gofra
var config gofra.Config
var mucs = make(map[string]jid.JID)
var client = &muc.Client{}
var occupants = make(map[string][]string)

func (p plugin) Name() string {
	return "MUC"
}

func (p plugin) Description() string {
	return "Handles multi user chat rooms"
}

func (p plugin) Help() string {
	return "MUC is a meta-plugin and does not expose user-triggered interaction"
}

func (p plugin) Init(conf gofra.Config, gofra *gofra.Gofra) {
	g = gofra
	config = conf
	prepareMUCs()
	g.Subscribe(
		"connected",
		p.Name(),
		joinMUCs,
		0,
	)
	g.Subscribe(
		"presenceReceived",
		p.Name(),
		handlePresence,
		0,
	)
	g.Subscribe(
		"muc/getOccupants",
		p.Name(),
		getOccupants,
		0,
	)
	g.AddMuxOption(muc.HandleClient(client))
}

func getOccupants(e gofra.Event) *gofra.Reply {
	return &gofra.Reply{Payload: map[string]interface{}{"occupants": occupants}}
}

func prepareMUCs() {
	if len(config.MUCs) == 0 {
		g.Logger.Warn(fmt.Sprintf("No MUCs in config: %v", config))

		return
	}

	for _, muc := range config.MUCs {
		occupants[muc.Jid] = []string{}
	}
}

func handlePresence(e gofra.Event) *gofra.Reply {
	pres, ok := e.GetStanza().(stanza.Presence)
	if !ok {
		g.Logger.Debug(fmt.Sprintf("Ignoring packet: %T\n", pres))

		return nil
	}

	occupantNick := ""
	// Parse presence and determine if it's MUC-related
	mucJid := pres.From.Bare().String()

	if pres.From.Resourcepart() != "" {
		occupantNick = pres.From.Resourcepart()
	}
	_, exists := occupants[mucJid]

	if !exists {

		return nil
	}

	if occupantNick == "" {

		return nil
	}

	if pres.Type == stanza.UnavailablePresence {
		if occupantLeft(mucJid, occupantNick) {
			g.Publish(
				gofra.Event{
					Name: "muc/occupantLeftMuc",
					Payload: map[string]interface{}{
						occupantNick: occupantNick,
						mucJid:       mucJid,
					},
				},
			)

			g.Publish(
				gofra.Event{
					Name: "muc/occupants",
					Payload: map[string]interface{}{
						"occupants": occupants,
					},
				},
			)
		}
	}

	if !occupantJoined(mucJid, occupantNick) {
		return nil
	}

	g.Publish(
		gofra.Event{
			Name: "muc/occupantJoinedMuc",
			Payload: map[string]interface{}{
				occupantNick: occupantNick,
				mucJid:       mucJid,
			},
		},
	)

	g.Publish(
		gofra.Event{
			Name: "muc/occupants",
			Payload: map[string]interface{}{
				"occupants": occupants,
			},
		},
	)

	return nil
}

func occupantLeft(room, occupant string) bool {
	position, exists := isOccupant(room, occupant)
	if !exists {

		return false
	}
	occupants[room][position] = occupants[room][len(room)-1]
	occupants[room] = occupants[room][:len(room)-1]

	return true
}

func occupantJoined(room, occupant string) bool {
	_, exists := isOccupant(room, occupant)
	if exists {
		return false
	}

	occupants[room] = append(occupants[room], occupant)

	return true
}

func isOccupant(room, occupant string) (int, bool) {
	position := -1
	for index, occ := range occupants[room] {
		if occ == occupant {
			position = index
			break
		}
	}

	return position, position != -1
}

func joinMUCs(e gofra.Event) *gofra.Reply {
	if len(config.MUCs) == 0 {
		return nil
	}

	for _, muc := range config.MUCs {
		joinMUC(muc)
	}

	return nil
}

func joinMUC(mc gofra.MUCConfig) {
	g.Logger.Debug("Tried to join room: " + mc.Jid)
	j := jid.MustParse(mc.Jid + "/" + mc.Nick)

	_, exists := mucs[mc.Jid]
	if exists {
		return
	}

	mucOpts := []muc.Option{}

	if mc.Nick != "" {
		mucOpts = append(mucOpts, muc.Nick(mc.Nick))
	} else {
		mucOpts = append(mucOpts, muc.Nick(config.Nick))
	}

	mucOpts = append(mucOpts, muc.MaxHistory(uint64(mc.JoinHistory)))

	if mc.Password != "" {
		mucOpts = append(mucOpts, muc.Password(mc.Password))
	}

	go func() {
		_, err := client.Join(g.Context, jid.MustParse(mc.Jid+"/"+mc.Nick), g.Client, mucOpts...)

		if err != nil {
			g.Logger.Error(fmt.Sprintf("error joining: %v", err))
		}

		e := gofra.Event{Name: "muc/joinedRoom", Payload: map[string]interface{}{"roomJid": mc.Jid}}

		occupantJoined(mc.Jid, mc.Nick)
		mucs[mc.Jid] = j

		g.Publish(e)
	}()
}
