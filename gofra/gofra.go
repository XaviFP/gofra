package gofra

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

var mucNicks map[string]string

// Interface providing plugins the needed tools to interact with the engine
// and/or other plugins
type API interface {
	SendMessage(to, message string, msgType stanza.MessageType) error
	Subscribe(eventName, pluginName string, handler Handler, priority int)
	SubscribeChain(eventName, pluginName string, handler ChainHandler, priority int)
	Publish(event Event) Reply
	SetPriority(eventName, pluginName string, priority int) error
	SendStanza(stanza interface{}) error
	AddMuxOption(o mux.Option)
	AddMuxOptions(opts []mux.Option)
}
type Gofra struct {
	config       Config
	em           EventManager
	plugins      Plugins
	Client       *xmpp.Session
	Context      context.Context
	Logger       Logger
	serveMux     *mux.ServeMux
	serveMuxOpts []mux.Option
	initialized  bool
}

func NewGofra(ctx context.Context, config Config) *Gofra {
	logger := NewLogger(config.Debug)
	xmlIn, xmlOut := getStreamLoggers(config.LogXML)

	c, err := newXmppClient(ctx, config, xmlIn, xmlOut, logger)
	if err != nil {
		log.Fatal(err.Error())
	}

	gofra := &Gofra{
		config:  config,
		em:      NewEventManager(logger),
		plugins: NewPlugins(config),
		Client:  c,
		Context: ctx,
		Logger:  logger,
	}

	stanzaHandler := stanzaHandler{
		logger: logger,
		publish: func(e Event) {
			gofra.Publish(e)
		},
	}

	gofra.serveMuxOpts = []mux.Option{
		mux.Presence(stanza.AvailablePresence, xml.Name{}, stanzaHandler),
		mux.Presence(stanza.UnavailablePresence, xml.Name{}, stanzaHandler),
		mux.Message(stanza.ChatMessage, xml.Name{Space: "jabber:client", Local: "body"}, stanzaHandler),
		mux.Message(stanza.GroupChatMessage, xml.Name{Space: "jabber:client", Local: "body"}, stanzaHandler),
	}

	mucNicks = make(map[string]string)
	for _, muc := range config.MUCs {
		mucNicks[muc.Jid] = muc.Nick
	}

	return gofra
}

// Wrapper to ease sending messages
func (g *Gofra) SendMessage(to, body string, msgType stanza.MessageType) error {
	j, err := jid.Parse(to)
	if err != nil {
		return err
	}

	msg := MessageBody{
		Message: stanza.Message{
			Type: msgType,
			To:   j.Bare(),
		},
		Body: body,
	}

	return g.Client.Encode(g.Context, msg)
}

func (g *Gofra) SendStanza(s interface{}) error {
	return g.Client.Encode(g.Context, s)
}

/*
Adds an event listener for a given event. Event listeners are executed in descending
priority order, so a higher priority grants earlier execution in the queue.
For accumulative handlers, that is, handlers that take the original set of values of
the event and pass on a modified set, there's the chain option. Handlers set to chain
are executed after all non-accumulative ones by descending priority order. Accumulated
event values are received through the event pointer argument where changes are expecteted
to be performed in order for the following chained handlers to recieve them. */
func (g *Gofra) Subscribe(eventName, pluginName string, handler Handler, priority int) {
	g.Logger.Debug("Plugin " + pluginName + " subscribed handler to event " + eventName)
	g.em.Subscribe(eventName, pluginName, handler, nil, priority)
}

// Subscribes a chained event listener to an event
func (g *Gofra) SubscribeChain(eventName, pluginName string, handler ChainHandler, priority int) {
	g.Logger.Debug("Plugin " + pluginName + " subscribed chained handler to event " + eventName)
	g.em.Subscribe(eventName, pluginName, nil, handler, priority)
}

// Publish executes all event handlers subscribed to a particular event
func (g *Gofra) Publish(event Event) *Reply {
	return g.em.Publish(event)
}

func (g *Gofra) SetPriority(eventName, pluginName string, priority int) error {
	return g.em.SetPriority(eventName, pluginName, priority)
}

func (g *Gofra) AddMuxOption(o mux.Option) {
	g.serveMuxOpts = append(g.serveMuxOpts, o)
}

func (g *Gofra) AddMuxOptions(opts []mux.Option) {
	g.serveMuxOpts = append(g.serveMuxOpts, opts...)
}

func (g *Gofra) Init() error {
	if g.initialized {
		return nil
	}

	g.initialized = true

	// Initialize plugins
	err := g.plugins.Init(g.config, g)
	if err != nil {
		return err
	}

	// Initialize stanza multiplexer after registering all plugin-specific routes
	g.serveMux = mux.New("jabber:client", g.serveMuxOpts...)

	g.Publish(Event{Name: "initialized"})

	return nil
}

func (g *Gofra) Connect() error {
	// Send initial presence
	err := g.Client.Send(g.Context, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		return fmt.Errorf("error sending initial presence: %w", err)
	}

	g.Publish(Event{Name: "connected"})

	return g.Client.Serve(xmpp.HandlerFunc(g.serveMux.HandleXMPP))
}

func newXmppClient(ctx context.Context, config Config, xmlIn, xmlOut io.Writer, logger Logger) (*xmpp.Session, error) {
	j, err := jid.Parse(config.Jid)
	if err != nil {
		return nil, fmt.Errorf("error parsing address %q: %w", config.Jid, err)
	}

	var d dial.Dialer
	if config.SkipSRV {
		d.NoLookup = true
	}

	conn, err := d.Dial(ctx, "tcp", j)
	if err != nil {
		return nil, fmt.Errorf("error dialing sesion: %w", err)
	}

	s, err := xmpp.NewSession(ctx, j.Domain(), j, conn, 0, xmpp.NewNegotiator(func(*xmpp.Session, *xmpp.StreamConfig) xmpp.StreamConfig {
		return xmpp.StreamConfig{
			Lang: "en",
			Features: []xmpp.StreamFeature{
				xmpp.BindResource(),
				xmpp.StartTLS(&tls.Config{
					ServerName: j.Domain().String(),
					MinVersion: tls.VersionTLS12,
				}),
				xmpp.SASL("", config.Password, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
			},
			TeeIn:  xmlIn,
			TeeOut: xmlOut,
		}
	}))
	if err != nil {
		return nil, fmt.Errorf("error establishing a session: %w", err)
	}

	return s, nil
}
