package gofra

import (
	"gosrc.io/xmpp/stanza"
)

type Plugin interface {
	Name() string
	Description() string
	Init(Config, API)
}

type Runnable interface {
	Run()
}

type API interface {
	Send(to, message string, msgType stanza.StanzaType) error
	Subscribe(eventName, pluginName string, handler Handler, options Options)
	Publish(event Event) Reply
	SetPriority(eventName, pluginName string, options Options) error
	SendStanza(stanza stanza.Packet) error
}

type Config struct {
	ServerURL string
	ServerPort string
	//Temporary
	Password string
	Plugins_paths []string
	Jid string
	Nick string
	Mucs []MucConfig
	MucJoinHistory int64
	Extra map[string]interface{}
}

type MucConfig struct {
	Nick string
	MucJoinHistory int
	MucJid string
}

type Send func(to, message string, msgType stanza.StanzaType) error

//////////////////// EVENTS ////////////////////
type Handler func(event Event, accumulated *Event) (Reply, Event)

type EventHandler struct {
	Handler Handler
	Priority int64
	PluginName string
	Chain bool
}

type Event struct {
	Name string
	Payload map[string]interface{}
	Stanza stanza.Packet
}

type Options struct {
	Priority int64
}

type Reply struct{
	Reply map[string]interface{}
	Ok bool
	Empty bool
} 

func (r *Reply) SetAnswer(answer string) {
	if r.Reply == nil {
		r.Reply = make(map[string]interface{})
	}
	r.Reply["answer"] = answer
}

func (r *Reply) GetAnswer() string {
	if r.Reply == nil {
		r.Reply = make(map[string]interface{})
	}
	answer, exists := r.Reply["answer"]
	if !exists {
		return ""
	}
	strAnswer, ok := answer.(string)
	if !ok {
		return ""
	}
	return strAnswer
}

func (r *Reply) SetNoHandlers(noHandlers bool) {
	if r.Reply == nil {
		r.Reply = make(map[string]interface{})
	}
	r.Reply["noHandlers"] = noHandlers
}

func (r *Reply) GetNoHandlers() bool {
	if r.Reply == nil {
		r.Reply = make(map[string]interface{})
	}
	noHandlers, exists := r.Reply["noHandlers"]
	if !exists {
		return false
	}
	noHandlersBool, ok := noHandlers.(bool)
	if !ok {
		return false
	}
	return noHandlersBool
}