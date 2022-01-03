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
	ServerURL   string                            `yaml:"serverUrl"`
	ServerPort  string                            `yaml:"serverPort"`
	Password    string                            `yaml:"password"`
	PluginPaths []string                          `yaml:"pluginPaths"`
	Jid         string                            `yaml:"jid"`
	Nick        string                            `yaml:"nick"`
	LogXML      bool                              `yaml:"logXML"`
	Verbose     bool                              `yaml:"verbose"`
	Mucs        []MucConfig                       `yaml:"mucs"`
	Plugins     map[string]map[string]interface{} `yaml:"plugins"`
	Extra       map[string]interface{}            `yaml:"extra"`
}

// Per-MUC configuration
type MucConfig struct {
	Nick        string `yaml:"mucNick"`
	JoinHistory int    `yaml:"mucJoinHistory"`
	Jid         string `yaml:"mucJid"`
	Password    string `yaml:"mucPasword"`
}

//////////////////// EVENTS /////////////////////
