/*
muc is a gofra plugin that allows joining muti-user chatrooms and keeps track of them
*/

package main

import (
	"fmt"
	"log"

	"gofra/gofra"
	"strings"

	"gosrc.io/xmpp/stanza"
)

type plugin string

var g gofra.API
var config gofra.Config
var mucs = make(map[string]string)
var joinedMucs = make(map[string][]string)

func (p plugin) Name() string {
	return "MUC"
}

func (p plugin) Description() string {
	return "Handles multi user chat rooms"
}

func (p plugin) Init(conf gofra.Config, api gofra.API) {
	g = api
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
}

func prepareMUCs () {
	if len(config.Mucs) == 0 {
		log.Printf("No MUCs in config: %v", config)
		return
	}
	for _, muc := range config.Mucs {
		mucs[muc.MucJid] = muc.Nick
	}
}

func handlePresence(e gofra.Event, _ *gofra.Event) (gofra.Reply, gofra.Event){
	pres, ok := e.Stanza.(stanza.Presence)
	if !ok {
		log.Println("Ignoring packet: %T\n", e.Stanza)
		return gofra.Reply{Empty: true}, e
	}
	occupantNick := ""
	//Parse presence and determine if it's MUC-related
	jid := strings.Split(pres.From, "/")
	mucJid := jid[0]
	if len(jid) > 1 {
		occupantNick = strings.Split(pres.From, "/")[1]
	}
	_, exists := mucs[mucJid]
	if !exists {
		log.Println("MUC " + mucJid + " not found in config")
		return gofra.Reply{Ok: false, Empty: true}, e
	}

	log.Println("Joined MUCs: ", joinedMucs)
	occupants, exist := joinedMucs[mucJid]
	if !exist {
		joinedMucs[mucJid] = []string{}
		g.Publish(
			gofra.Event{
				Name: "mucJoined",
		})
	}
	if occupantNick == "" {
		return gofra.Reply{Ok: true, Empty: true}, e
	}
	if !pres.Attrs.Type.IsEmpty() && pres.Type != stanza.PresenceTypeUnavailable {
		leaveRoom(joinedMucs[mucJid], occupantNick)
		g.Publish(
			gofra.Event{
				Name: "occupantLeftMuc",
		})
	}

	alreadyIn := false
	for _, occupant := range occupants {
		if occupant == occupantNick {
			alreadyIn = true
			break
		}
	}
	if alreadyIn {
		return gofra.Reply{Ok: true, Empty: true}, e
	}
	if !joinRoom(joinedMucs[mucJid], occupantNick) {
		return gofra.Reply{Ok: true, Empty: true}, e
	}
	g.Publish(
		gofra.Event{
			Name: "occupantJoinedMuc",
	})
	return gofra.Reply{Ok: true, Empty: true}, e
}

func leaveRoom(room []string, occupant string) {
	position, exists := isOccupant(room, occupant)
	if !exists {
		return
	}
	room[position] = room[len(room)-1]
    room = room[:len(room)-1]
}

func joinRoom(room[]string, occupant string) bool{
	_, exists := isOccupant(room, occupant)
	if exists {
		return false
	}
	room = append(room, occupant)
	return true
}

func isOccupant(room []string, occupant string) (int, bool) {
	position := -1
	for index, occ := range room {
		if occ == occupant {
			position = index
			break
		}
	}
	return position, position != -1
}

func joinMUCs(e gofra.Event, _ *gofra.Event) (gofra.Reply, gofra.Event){
	if len(config.Mucs) == 0 {
		return gofra.Reply{Empty: true}, e
	}
	for _, muc := range config.Mucs {
		joinMUC(muc)
	}
	return gofra.Reply{Empty: true}, e
}

func joinMUC(mc gofra.MucConfig){
	//Join MUC
	mucJoinPres := mucJoinPresence(config.Jid, mc.Nick, mc.MucJid, mc.MucJoinHistory)
	err := g.SendStanza(mucJoinPres)
	if err != nil {
		log.Println(err)
		fmt.Printf("Couldn't send presence stanza to join muc %s", mc.MucJid)
	}
}

func mucJoinPresence(selfJid, nick, mucJid string, mucJoinHistory int) *stanza.Presence {
	presenceStanza := stanza.Presence{
		Attrs: stanza.Attrs{
			To:   mucJid + "/" + nick,
			From: selfJid,
		},
		Extensions: []stanza.PresExtension{
			stanza.MucPresence{
				History: stanza.History{MaxStanzas: stanza.NewNullableInt(mucJoinHistory)},
			},
		},
	}
	return &presenceStanza
} 

var Plugin plugin