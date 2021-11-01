package gofra

import (
	"encoding/json"
	"log"
	"gosrc.io/xmpp/stanza"
)

// Interface to be satisfied by any gofra plugin
type Plugin interface {
	Name() string
	Description() string
	Init(Config, API)
}

// Interface to be satisfied by plugins that need an execution loop
// like, for example, an HTTP server. Run method is executed as a goroutine.
type Runnable interface {
	Run()
}

// Interface providing plugins the needed tools to interact with the engine
// and/or other plugins
type API interface {
	Send(to, message string, msgType stanza.StanzaType) error
	Subscribe(eventName, pluginName string, handler Handler, options Options)
	Publish(event Event) Reply
	SetPriority(eventName, pluginName string, options Options) error
	SendStanza(stanza stanza.Packet) error
}

type Config struct {
	ServerURL string `yaml:"serverUrl"`
	ServerPort string`yaml:"serverUrl"`
	Password string `yaml:"password"`
	PluginPaths []string `yaml:"pluginPaths"`
	Jid string `yaml:"jid"`
	Nick string `yaml:"nick"`
	Mucs []MucConfig `yaml:"mucConfigs"`
	Plugins map[string]interface{} `yaml:"plugins"`
	Extra map[string]interface{} `yaml:"extra"`
}

// Per-MUC configuration
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
	Name string `json:"name"`
	Payload map[string]interface{} `json:"payload"`
	Stanza stanza.Packet `json:"stanza"`
}

type Options struct {
	Priority int64
	Chain bool
}

type Reply struct{
	Payload map[string]interface{}
	Ok bool
	Empty bool
} 

// Clone deepcopies a to b using json marshaling
func Clone(a, b interface{}) {
    bytes, err := json.Marshal(a)
	if err != nil {
		log.Print(err)
	}
    err = json.Unmarshal(bytes, b)
	if err != nil {
		log.Print(err)
	}
}

// Data access interface for text-based commands to answer to a suitable message.
func (r *Reply) SetAnswer(answer string) {
	if r.Payload == nil {
		r.Payload = make(map[string]interface{})
	}
	r.Payload["answer"] = answer
}

// Data access interface for command plugin to receive the answer from an specific command.
func (r *Reply) GetAnswer() string {
	if r.Payload == nil {
		r.Payload = make(map[string]interface{})
	}
	answer, exists := r.Payload["answer"]
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
	if r.Payload == nil {
		r.Payload = make(map[string]interface{})
	}
	r.Payload["noHandlers"] = noHandlers
}

func (r *Reply) GetNoHandlers() bool {
	if r.Payload == nil {
		r.Payload = make(map[string]interface{})
	}
	noHandlers, exists := r.Payload["noHandlers"]
	if !exists {
		return false
	}
	noHandlersBool, ok := noHandlers.(bool)
	if !ok {
		return false
	}
	return noHandlersBool
}