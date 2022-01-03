/*
muc is a gofra plugin that allows joining muti-user chatrooms and keeps track of them
*/

package main

import (
	"gofra/gofra"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/stanza"
)

type plugin string

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

func (p plugin) Init(conf gofra.Config, api gofra.API) {
	g = api.(*gofra.Gofra)
	config = conf
	prepareMUCs()
	g.Subscribe(
		"connected",
		p.Name(),
		joinMUCs,
		gofra.Options{},
	)
	g.Subscribe(
		"presenceReceived",
		p.Name(),
		handlePresence,
		gofra.Options{},
	)
	g.AddMuxOption(muc.HandleClient(client))
}

func prepareMUCs() {
	if len(config.Mucs) == 0 {
		g.Logger.Printf("No MUCs in config: %v", config)
		return
	}
	for _, muc := range config.Mucs {
		occupants[muc.Jid] = []string{}
	}
}

func handlePresence(e gofra.Event) (gofra.Reply){
	pres, ok := e.GetStanza().(stanza.Presence)
	if !ok {
		g.Logger.Println("Ignoring packet: %T\n", pres)
		return gofra.Reply{Empty: true}
	}

	occupantNick := ""
	//Parse presence and determine if it's MUC-related
	mucJid := pres.From.Bare().String()

	if pres.From.Resourcepart() != "" {
		occupantNick = pres.From.Resourcepart()
	}
	_, exists := occupants[mucJid]

	if !exists {
		g.Logger.Println("MUC " + mucJid + " not found in config")
		return gofra.Reply{Ok: false, Empty: true}
	}

	if occupantNick == "" {
		return gofra.Reply{Ok: true, Empty: true}
	}

	if pres.Type == stanza.UnavailablePresence {
		occupantLeft(mucJid, occupantNick)
		g.Publish(
			gofra.Event{
				//TODO send muc and occupant in the event
				Name: "muc/occupantLeftMuc",
		})
	}

	if !occupantJoined(mucJid, occupantNick) {
		return gofra.Reply{Ok: true, Empty: true}
	}

	g.Publish(
		gofra.Event{
			//TODO send muc and occupant in the event
			Name: "muc/occupantJoinedMuc",
	})
	return gofra.Reply{Ok: true, Empty: true}
}

func occupantLeft(room, occupant string) {
	position, exists := isOccupant(room, occupant)
	if !exists {
		return
	}
	occupants[room][position] = occupants[room][len(room)-1]
    occupants[room] = occupants[room][:len(room)-1]
}

func occupantJoined(room, occupant string) bool{
	_, exists := isOccupant(room, occupant)
	if exists {
		return false
	}
	occupants[room] = append(occupants[room], occupant)
	g.Logger.Println(occupants)
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

func joinMUCs(e gofra.Event) (gofra.Reply){
	if len(config.Mucs) == 0 {
		return gofra.Reply{Empty: true}
	}
	for _, muc := range config.Mucs {
		joinMUC(muc)
	}
	return gofra.Reply{Empty: true}
}

func joinMUC(mc gofra.MucConfig){
	g.Logger.Println("Tried to join room: " + mc.Jid)
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
		_, err := client.Join(g.Context, jid.MustParse(mc.Jid + "/" + mc.Nick), g.Client, mucOpts...)

		if err != nil {
			g.Logger.Fatalf("error joining: %v", err)
		}

		e := gofra.Event{Name: "muc/joinedRoom", Payload: map[string]interface{}{"roomJid": mc.Jid}}

		occupantJoined(mc.Jid, mc.Nick)
		mucs[mc.Jid] = j

		g.Publish(e)
	}()
}

var Plugin plugin