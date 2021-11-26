package gofra

import (
	"mellium.im/xmpp/stanza"
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
	SendMessage(to, message string, msgType stanza.MessageType) error
	Subscribe(eventName, pluginName string, handler Handler, options Options)
	SubscribeChain(eventName, pluginName string, handler ChainHandler, options Options)
	Publish(event Event) Reply
	SetPriority(eventName, pluginName string, options Options) error
	SendStanza(stanza interface{}) error
}

type Config struct {
	ServerURL string `yaml:"serverUrl"`
	ServerPort string`yaml:"serverPort"`
	Password string `yaml:"password"`
	PluginPaths []string `yaml:"pluginPaths"`
	Jid string `yaml:"jid"`
	Nick string `yaml:"nick"`
	LogXML bool `yaml:"logXML"`
	Verbose bool `yaml:"verbose"`
	Mucs []MucConfig `yaml:"mucs"`
	Plugins map[string]map[string]interface{} `yaml:"plugins"`
	Extra map[string]interface{} `yaml:"extra"`
}

// Per-MUC configuration
type MucConfig struct {
	Nick string `yaml:"mucNick"`
	MucJoinHistory int `yaml:"mucJoinHistory"`
	MucJid string `yaml:"mucJid"`
}

//////////////////// EVENTS /////////////////////

type Handler func(event Event) Reply
type ChainHandler func(accumulated *Event)

type EventHandler struct {
	Handler Handler
	Priority int64
	PluginName string
	Chain ChainHandler
}

type Event struct {
	Name string
	Payload map[string]interface{}
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

func (e *Event) SetStanza(stanza interface{}) {
	if e.Payload == nil {
		e.Payload = make(map[string]interface{})
	}
	e.Payload["stanza"] = stanza
}

func (e *Event) GetStanza() interface{} {
	if e.Payload == nil {
		e.Payload = make(map[string]interface{})
	}
	stanza, exists := e.Payload["stanza"]
	if !exists {
		return nil
	}
	return stanza
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