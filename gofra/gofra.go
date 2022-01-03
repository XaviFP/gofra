package gofra

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

type Gofra struct {
	config  Config
	events  Events
	plugins Plugins
	Client  *xmpp.Session
	Context context.Context
	Logger  *log.Logger
	Debug   *log.Logger
	mux     *mux.ServeMux // TODO: Rename to ServeMux
	opts    []mux.Option  // TODO: muxOpts or more specific name (currently it looks like Gofra opts, not ServeMux opts)
}

var gofra *Gofra
var initialized bool

func NewGofra(ctx context.Context, config Config, xmlIn, xmlOut *io.Writer, logger, debug *log.Logger) *Gofra {
	if gofra != nil {
		return gofra
	}

	c, err := newXmppClient(ctx, config, xmlIn, xmlOut, logger, debug)
	if err != nil {
		log.Fatal(err.Error())
	}

	opts := []mux.Option{
		mux.Presence(stanza.AvailablePresence, xml.Name{}, stanzaHandler{}),
		mux.Presence(stanza.UnavailablePresence, xml.Name{}, stanzaHandler{}),
		mux.Message(stanza.ChatMessage, xml.Name{Space: "jabber:client", Local: "body"}, stanzaHandler{}),
		mux.Message(stanza.GroupChatMessage, xml.Name{Space: "jabber:client", Local: "body"}, stanzaHandler{}),
	}

	gofra = &Gofra{
		config:  config,
		events:  NewEvents(config),
		plugins: NewPlugins(config),
		Client:  c,
		Context: ctx,
		Logger:  logger,
		Debug:   debug,
		// opts:    opts, // TODO check if this is possible to avoid the AddMuxOptions
	}

	gofra.AddMuxOptions(opts)

	return gofra
}

///////////////////// API ///////////////////////

// Send function wrapper to make sending messages easier
func (g *Gofra) SendMessage(to, body string, msgType stanza.MessageType) error {
	j, err := jid.Parse(to)
	if err != nil {
		return err
	}
	msg := MessageBody{Message: stanza.Message{Type: msgType, To: j.Bare()}, Body: body}
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
func (g *Gofra) Subscribe(eventName, pluginName string, handler Handler, options Options) {
	g.Logger.Println("Plugin " + pluginName + " subscribed handler to event " + eventName)
	g.events.Subscribe(eventName, pluginName, handler, nil, options)
}

// TODO change naming and method comments between Subscribe and SubscribeChain
func (g *Gofra) SubscribeChain(eventName, pluginName string, handler ChainHandler, options Options) {
	g.Logger.Println("Plugin " + pluginName + " subscribed chained handler to event " + eventName)
	g.events.Subscribe(eventName, pluginName, nil, handler, options)
}

// Executes all event handlers subscribed to a particular event
func (g *Gofra) Publish(event Event) Reply {
	return g.events.Publish(event)
}

func (g *Gofra) SetPriority(eventName, pluginName string, options Options) error {
	return g.events.SetPriority(eventName, pluginName, options)
}

func (g *Gofra) AddMuxOption(o mux.Option) {
	g.opts = append(g.opts, o)
}

func (g *Gofra) AddMuxOptions(opts []mux.Option) {
	g.opts = append(g.opts, opts...)
}

/////////////////////////////////////////////////

func (g *Gofra) Init() error {
	if initialized {
		return nil
	}

	initialized = true

	// Initialize plugins
	err := g.plugins.Init(g.config, g)
	if err != nil {
		return err
	}

	// Initialize stanza multiplexer after registering all plugin-specific routes
	g.mux = mux.New("jabber:client", g.opts...)

	g.Publish(Event{Name: "initialized"})

	return nil
}

func (g *Gofra) Connect() error {
	// Send initial presence
	err := g.Client.Send(gofra.Context, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		return fmt.Errorf("error sending initial presence: %w", err)
	}

	g.Publish(Event{Name: "connected"})

	return gofra.Client.Serve(xmpp.HandlerFunc(g.mux.HandleXMPP))
}

type logWriter struct {
	logger *log.Logger
}

func (lw logWriter) Write(p []byte) (int, error) {
	lw.logger.Printf("%s", p)

	return len(p), nil
}

func newXmppClient(ctx context.Context, config Config, xmlIn, xmlOut *io.Writer, logger, debug *log.Logger) (*xmpp.Session, error) {
	j, err := jid.Parse(config.Jid)
	if err != nil {
		return nil, fmt.Errorf("error parsing address %q: %w", config.Jid, err)
	}
	// TODO Remember to remove workaround before publishing the project
	var d dial.Dialer

	d.NoLookup = true
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
			TeeIn:  logWriter{log.New(os.Stdout, "IN ", log.LstdFlags)},
			TeeOut: logWriter{log.New(os.Stdout, "OUT ", log.LstdFlags)},
		}
	}))
	if err != nil {
		return nil, fmt.Errorf("error establishing a session: %w", err)
	}

	return s, nil
}
